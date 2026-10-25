package face

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andyzhou/websocket/define"
	"github.com/andyzhou/websocket/gvar"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * websocket connect face
 */

//connect config
type ConnConf struct {
	BucketId       int
	GroupId        int64
	AsyncWorkerNum int
	MessageType    int
	CBForClosed    func(connId int64) error
	CBForRead      func(connId int64, messageType int, data interface{}) error
}

//face info
type Connector struct {
	conf             *ConnConf
	connId           int64 //origin conn id
	ownerId          int64
	activeTime       int64
	conn             *websocket.Conn //origin conn reference
	propertyMap      map[string]interface{}
	writeChan        chan interWriteData //write byte chan
	writeJsonChan    chan interface{}
	closeChan        chan bool
	messageChan      chan interface{} //async message chan
	messageCloseChan chan bool
	asyncWorkerNum   int //async worker number
	readTimeout      time.Duration
	writeTimeout     time.Duration
	readDeadline     time.Time
	writeDeadline    time.Time
	closeOnce        sync.Once
	closed 			 atomic.Bool
	propLocker       sync.RWMutex
	connLocker       sync.RWMutex
	deadlineLocker   sync.RWMutex
	Util
}

type interWriteData struct {
	data 		interface{}
	messageType int
}

//construct
//timeouts=> readTimeout, writeTimeout
func NewConnector(
	conf *ConnConf,
	connId int64,
	conn *websocket.Conn,
	timeouts ...time.Duration) *Connector {
	this := &Connector{
		conf:             conf,
		connId:           connId,
		conn:             conn,
		writeChan:        make(chan interWriteData, define.ConnWriteChanSize),
		writeJsonChan:    make(chan interface{}, define.ConnWriteChanSize),
		closeChan:        make(chan bool, 2),
		propertyMap:      map[string]interface{}{},
		messageChan:      make(chan interface{}, define.MessageChanSize),
		messageCloseChan: make(chan bool, 1),
		asyncWorkerNum:   define.ASyncWorkerNum,
	}
	this.interInit(timeouts...)
	return this
}

//close
//close
func (f *Connector) Close() {
	f.closeOnce.Do(func() {
		f.closed.Store(true)

		// 1) 先通知各 loop 退出，避免 chan 关闭后空转
		close(f.messageCloseChan)

		// 2) 关闭底层连接（在锁内），并关闭 closeChan
		f.connLocker.Lock()
		if f.conn != nil {
			close(f.closeChan)
			f.conn.Close()
			f.conn = nil
		} else {
			// 即使 conn 为 nil，也要关 closeChan，保证 writeProcess 能退出
			// 注意：closeChan 只会被 close 一次（closeOnce 保护）
			close(f.closeChan)
		}
		f.connLocker.Unlock()

		// 3) 最后关闭 writeChan（此时 writeProcess 已能通过 closeChan 退出）
		close(f.writeChan)
	})
}

//set config id
func (f *Connector) SetConfId(bucketId int, groupId int64) {
	f.conf.BucketId = bucketId
	f.conf.GroupId = groupId
}

//get active time
func (f *Connector) GetActiveTime() int64 {
	return f.activeTime
}

//get owner id
func (f *Connector) GetOwnerId() int64 {
	return f.ownerId
}

//set owner id
func (f *Connector) SetOwnerId(ownerId int64) {
	f.ownerId = ownerId
}

//remove property
func (f *Connector) RemoveProp(kind string) error {
	//check
	if kind == "" {
		return errors.New("invalid parameter")
	}

	//remove with locker
	f.propLocker.Lock()
	defer f.propLocker.Unlock()
	delete(f.propertyMap, kind)
	return nil
}

//get or set property
func (f *Connector) GetProp(kind string) (interface{}, error) {
	//check
	if kind == "" {
		return nil, errors.New("invalid parameter")
	}

	//get with locker
	f.propLocker.RLock()
	defer f.propLocker.RUnlock()
	v, ok := f.propertyMap[kind]
	if ok && v != nil {
		return v, nil
	}
	return nil, errors.New("no such property")
}

func (f *Connector) SetProp(kind string, val interface{}) error {
	//check
	if kind == "" || val == nil {
		return errors.New("invalid parameter")
	}

	//save with locker
	f.propLocker.Lock()
	defer f.propLocker.Unlock()
	f.propertyMap[kind] = val
	return nil
}

//close with message
func (f *Connector) CloseWithMessage(message string) error {
	//check
	if message == "" {
		return errors.New("invalid parameter")
	}
	if f.conn == nil {
		return errors.New("connect has closed")
	}

	// 写消息（只写一次，用 once 单独控制写消息这个动作）
	f.closeOnce.Do(func() {
		f.closed.Store(true)

		// 写最后一条消息
		f.connLocker.Lock()
		if f.conn != nil {
			f.conn.Write([]byte(message))
		}
		f.connLocker.Unlock()

		// 复用统一的关闭流程
		close(f.messageCloseChan)

		f.connLocker.Lock()
		if f.conn != nil {
			close(f.closeChan)
			f.conn.Close()
			f.conn = nil
		} else {
			close(f.closeChan)
		}
		f.connLocker.Unlock()

		close(f.writeChan)
	})
	return nil
}

//get origin connect id
func (f *Connector) GetConnId() int64 {
	return f.connId
}

//get uri paras
func (f *Connector) GetUriParas() map[string]string {
	if f.conn == nil {
		return nil
	}
	return mux.Vars(f.conn.Request())
}

func (f *Connector) GetUriQueryParas() url.Values {
	if f.conn == nil {
		return nil
	}
	return f.conn.Request().URL.Query()
}

//get origin connect reference
func (f *Connector) GetConn() *websocket.Conn {
	f.connLocker.RLock()
	defer f.connLocker.RUnlock()
	return f.conn
}

//push to write queue
//if directWrite is true, use low tcp to write bytes data
//or use json MessageTypeOfOctet mod
func (f *Connector) QueueWrite(data interface{}, messageTypes ...int) (err error) {
	var (
		messageType int
		m           any = nil
	)
	if data == nil {
		return errors.New("invalid parameter")
	}
	if f.closed.Load() {
		return fmt.Errorf("connect %v write chan is closed", f.connId)
	}
	if len(messageTypes) > 0 {
		messageType = messageTypes[0]
	}

	defer func() {
		if r := recover(); r != m {
			err = fmt.Errorf("write to closed channel: %v", r)
		}
	}()

	iwd := interWriteData{
		data:        data,
		messageType: messageType,
	}

	select {
	case f.writeChan <- iwd:
		return nil
	default:
		return fmt.Errorf("write channel unavailable")
	}
}

func (f *Connector) QueueWriteOld(data interface{}, messageTypes ...int) error {
	var (
		messageType int
		m any = nil
		err error
	)
	//check
	if data == nil {
		return errors.New("invalid parameter")
	}

	// 使用 atomic 检查
	if f.closed.Load() {
		return fmt.Errorf("connect %v write chan is closed", f.connId)
	}

	if len(messageTypes) > 0 {
		messageType = messageTypes[0]
	}

	// 用defer捕获可能的panic
	defer func() {
		if r := recover(); r != m {
			err = fmt.Errorf("write to closed channel: %v", r)
		}
	}()

	//isClosed, err := f.IsChanClosed(f.writeChan)
	//if err != nil {
	//	return err
	//}
	//if isClosed {
	//	return fmt.Errorf("connect %v write chan is closed", f.connId)
	//}

	//write to chan
	iwd := interWriteData {
		data: data,
		messageType: messageType,
	}

	// 即使这里检查通过，仍可能被关闭
	// 但结合 closeOnce，可以保证不会 panic
	select {
	case f.writeChan <- iwd:
		return nil
	default:
		// channel 可能已满或已关闭
		return fmt.Errorf("write channel unavailable")
	}
	return err
}

//send message with timeout
func (f *Connector) Write(data interface{}, messageTypes ...int) error {
	var (
		messageType int
		err error
	)
	//check
	if data == nil {
		return errors.New("invalid parameter")
	}
	if f.conn == nil {
		return errors.New("connect is nil")
	}
	if messageTypes != nil && len(messageTypes) > 0 {
		messageType = messageTypes[0]
	}

	//update active time
	defer func() {
		f.updateActiveTime(time.Now().Unix())
	}()

	//set write deadline
	f.connLocker.Lock()
	if f.conn == nil {
		f.connLocker.Unlock()
		return errors.New("connect is nil")
	}
	f.conn.SetWriteDeadline(time.Now().Add(f.writeTimeout))
	conn := f.conn
	f.connLocker.Unlock()

	//send real data
	switch messageType {
	case gvar.MessageTypeOfJson:
		{
			//json format
			err = websocket.JSON.Send(conn, data)
			if err != nil {
				log.Println(err)
			}
		}
	case gvar.MessageTypeOfOctet:
		fallthrough
	default:
		{
			//general octet format
			err = websocket.Message.Send(conn, data)
		}
	}
	return err
}

//read message with timeout
func (f *Connector) Read(messageTypes ...int) (interface{}, error) {
	var (
		messageType int
		err         error
		m           any
	)
	if messageTypes != nil && len(messageTypes) > 0 {
		messageType = messageTypes[0]
	}

	defer func() {
		f.updateActiveTime(time.Now().Unix())
		if pErr := recover(); pErr != m {
			log.Printf("connector.read panic, err:%v", pErr)
		}
	}()

	f.connLocker.RLock()
	if f.conn == nil {
		f.connLocker.RUnlock()
		return nil, errors.New("connect is nil")
	}
	conn := f.conn

	f.deadlineLocker.RLock()
	readDeadline := f.readDeadline
	f.deadlineLocker.RUnlock()

	// 仅当 deadline 已过期才重设，且 readTimeout 必须 > 0
	if time.Now().After(readDeadline) && f.readTimeout > 0 {
		f.deadlineLocker.Lock()
		f.readDeadline = time.Now().Add(f.readTimeout)
		_ = conn.SetReadDeadline(f.readDeadline)
		f.deadlineLocker.Unlock()
	}
	f.connLocker.RUnlock()

	switch messageType {
	case gvar.MessageTypeOfJson:
		var data interface{}
		err = websocket.JSON.Receive(conn, &data)
		return data, err
	case gvar.MessageTypeOfOctet:
		fallthrough
	default:
		var data []byte
		err = websocket.Message.Receive(conn, &data)
		return data, err
	}
}

func (f *Connector) ReadOld(messageTypes ...int) (interface{}, error) {
	var (
		messageType int
		err error
		m any
	)
	//check
	if messageTypes != nil && len(messageTypes) > 0 {
		messageType = messageTypes[0]
	}

	//update active time
	defer func() {
		f.updateActiveTime(time.Now().Unix())
		if pErr := recover(); pErr != m {
			log.Printf("connector.read panic, err:%v", pErr)
		}
	}()

	//get connector reference
	f.connLocker.RLock()
	if f.conn == nil {
		f.connLocker.RUnlock()
		return nil, errors.New("connect is nil")
	}
	conn := f.conn

	// 设置读取超时（减少锁范围）
	f.deadlineLocker.RLock()
	readDeadline := f.readDeadline
	f.deadlineLocker.RUnlock()

	if time.Now().After(readDeadline) {
		f.deadlineLocker.Lock()
		f.readDeadline = time.Now().Add(f.readTimeout)
		conn.SetReadDeadline(f.readDeadline)
		f.deadlineLocker.Unlock()
	}
	f.connLocker.RUnlock()

	//receive data
	switch messageType {
	case gvar.MessageTypeOfJson:
		{
			//json format
			var data interface{}
			err = websocket.JSON.Receive(conn, &data)
			return data, err
		}
	case gvar.MessageTypeOfOctet:
		fallthrough
	default:
		{
			//general octet format
			var data []byte
			err = websocket.Message.Receive(conn, &data)
			return data, err
		}
	}
}

//check connected or not
func (f *Connector) isConnected() bool {
	f.connLocker.RLock()
	defer f.connLocker.RUnlock()
	return f.conn != nil
}

//update active time
func (f *Connector) updateActiveTime(ts int64) {
	f.activeTime = ts
}

//write pure data
func (f *Connector) writePureData(data []byte) error {
	//check
	if data == nil {
		return errors.New("invalid parameter")
	}

	//access connect with locker
	f.connLocker.Lock()
	if f.conn == nil {
		//conn has closed
		return errors.New("conn is nil")
	}
	//setup write deadline
	f.conn.SetWriteDeadline(time.Now().Add(f.writeTimeout))
	conn := f.conn
	f.connLocker.Unlock()

	//write data
	_, err := conn.Write(data)
	if err != nil {
		if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
			//connect closed
			if f.conf != nil && f.conf.CBForClosed != nil {
				f.conf.CBForClosed(f.connId)
			}
			return err
		}
		log.Printf("connect %v write error: %v", f.connId, err)
	}
	return err
}

func (f *Connector) writeProcess() {
	var (
		iwd interWriteData
		isOk bool
	)
	////defer opt
	//defer func() {
	//	if !f.closed.Load() {
	//		close(f.writeChan)
	//	}
	//}()

	//loop
	for {
		select {
		case iwd, isOk = <- f.writeChan:
			{
				if !isOk {
					return
				}
				if &iwd != nil {
					f.Write(iwd.data, iwd.messageType)
				}
			}
		case <- f.closeChan:
			{
				return
			}
		}
	}
}

//read process
func (f *Connector) readProcess() {
	var (
		m                any = nil
		timeoutCount     int
		maxTimeoutCount  = 100   // 连续超时阈值，可按需调整
	)
	defer func() {
		if pErr := recover(); pErr != m {
			log.Printf("connect %v read process panic, err:%v\n", f.connId, pErr)
		}
		if !f.closed.Load() {
			// 非主动关闭时，确保清理（关闭所有 chan + conn）
			f.Close()
		}
	}()

	//start async message worker
	go f.asyncMessageWorker()

	for {
		if !f.isConnected() {
			return
		}

		data, err := f.Read(f.conf.MessageType)
		if err != nil {
			if err == io.EOF {
				if f.conf.CBForClosed != nil {
					f.conf.CBForClosed(f.GetConnId())
				}
				return
			}
			if netErr, sok := err.(net.Error); sok && netErr.Timeout() {
				timeoutCount++
				// 连续超时过多，判定连接已死，主动关闭
				if timeoutCount >= maxTimeoutCount {
					if f.conf.CBForClosed != nil {
						f.conf.CBForClosed(f.GetConnId())
					}
					return
				}
				// 退避，避免立即超时空转
				time.Sleep(10 * time.Millisecond)
				continue
			}
			// 其他错误（含 use of closed network connection），退出
			return
		}

		// 读到数据，重置超时计数
		timeoutCount = 0

		select {
		case f.messageChan <- data:
		default:
			log.Printf("connector %v message queue full, dropping message", f.connId)
		}
	}
}

func (f *Connector) readProcessOld() {
	var (
		m any = nil
	)
	defer func() {
		if pErr := recover(); pErr != m {
			log.Printf("connect %v read process panic, err:%v\n", f.connId, pErr)
		}
		if !f.closed.Load() {
			//close message chan
			close(f.messageChan)
		}
	}()

	//start async message worker
	go f.asyncMessageWorker()

	//read process loop
	for {
		if !f.isConnected() {
			//fix empty loop when lost connect
			return
		}
		data, err := f.Read(f.conf.MessageType)
		if err != nil {
			if err == io.EOF {
				if f.conf.CBForClosed != nil {
					f.conf.CBForClosed(f.GetConnId())
				}
				return
			}
			if netErr, sok := err.(net.Error); sok && netErr.Timeout() {
				//read timeout, but continue
				continue
			}
			//other error, just return
			return
		}

		//async process message
		select {
		case f.messageChan <- data:
		default:
			//queue full
			log.Printf("connector %v message queue full, dropping message", f.connId)
		}
	}
}

//async message worker
func (f *Connector) asyncMessageWorker() {
	var (
		data interface{}
		isOk bool
		m    any = nil
	)
	defer func() {
		if r := recover(); r != m {
			log.Printf("asyncMessageWorker panic: %v", r)
		}
	}()

	for {
		select {
		case data, isOk = <-f.messageChan:
			if !isOk {
				// messageChan 已关闭，退出，避免空转
				return
			}
			if f.conf != nil && f.conf.CBForRead != nil {
				func(connId int64, messageType int, data interface{}) {
					defer func() {
						if r := recover(); r != m {
							log.Printf("CBForRead panic: %v", r)
						}
					}()
					f.conf.CBForRead(connId, messageType, data)
				}(f.GetConnId(), f.conf.MessageType, data)
			}
		case <-f.messageCloseChan:
			return
		}
	}
}

func (f *Connector) asyncMessageWorkerOld() {
	var(
		data interface{}
		isOk bool
		m any = nil
	)
	defer func() {
		if r := recover(); r != m {
			log.Printf("asyncMessageWorker panic: %v", r)
		}
	}()

	//loop
	for {
		select {
		case data, isOk = <- f.messageChan:
			if isOk && &data != nil {
				if f.conf != nil && f.conf.CBForRead != nil {
					//unblock read data
					func(connId int64, messageType int, data interface{}) {
						defer func() {
							if r := recover(); r != m {
								log.Printf("CBForRead panic: %v", r)
							}
						}()
						f.conf.CBForRead(connId, messageType, data)
					}(f.GetConnId(), f.conf.MessageType, data)
				}
				break
			}
		case <- f.messageCloseChan:
			{
				return
			}
		}
	}
}

//inter init
func (f *Connector) interInit(timeouts ...time.Duration) {
	//setup default timeout
	f.readTimeout = 30 * time.Second
	f.writeTimeout = 10 * time.Second

	//overwrite timeout config from outside
	timeoutsLen := len(timeouts)
	switch timeoutsLen {
	case 1:
		f.readTimeout = timeouts[0]
	case 2:
		f.readTimeout = timeouts[0]
		f.writeTimeout = timeouts[1]
	}

	//setup async worker num
	if f.conf.AsyncWorkerNum > 0 {
		f.asyncWorkerNum = f.conf.AsyncWorkerNum
	}

	//init deadline
	f.deadlineLocker.Lock()
	f.readDeadline = time.Now().Add(f.readTimeout)
	f.writeDeadline = time.Now().Add(f.writeTimeout)
	f.deadlineLocker.Unlock()

	//update active time
	f.updateActiveTime(time.Now().Unix())

	//run write process
	go f.writeProcess()

	//run read process
	go f.readProcess()
}