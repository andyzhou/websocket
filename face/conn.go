package face

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"sync"
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
	writeChan        chan []byte //write byte chan
	closeChan        chan bool
	messageChan      chan interface{} //async message chan
	messageCloseChan chan bool
	asyncWorkerNum   int //async worker number
	readTimeout      time.Duration
	writeTimeout     time.Duration
	readDeadline     time.Time
	writeDeadline    time.Time
	closeOnce        sync.Once
	propLocker       sync.RWMutex
	connLocker       sync.RWMutex
	deadlineLocker   sync.RWMutex
	Util
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
		writeChan:        make(chan []byte, define.ConnWriteChanSize),
		closeChan:        make(chan bool, 1),
		propertyMap:      map[string]interface{}{},
		messageChan:      make(chan interface{}, define.MessageChanSize),
		messageCloseChan: make(chan bool, 1),
		asyncWorkerNum:   define.ASyncWorkerNum,
	}
	this.interInit(timeouts...)
	return this
}

//close
func (f *Connector) Close() {
	f.closeOnce.Do(func() {
		close(f.messageCloseChan)
		f.connLocker.Lock()
		defer f.connLocker.Unlock()
		if f.conn != nil {
			close(f.closeChan)
			f.conn.Close()
			f.conn = nil
		}
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

	//write message before close it
	f.closeOnce.Do(func() {
		close(f.messageCloseChan)
		f.connLocker.Lock()
		defer f.connLocker.Unlock()
		if f.conn != nil {
			f.conn.Write([]byte(message))
			f.conn.Close()
			f.conn = nil
		}
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
func (f *Connector) QueueWrite(data []byte) error {
	//check
	if data == nil {
		return errors.New("invalid parameter")
	}
	isClosed, err := f.IsChanClosed(f.writeChan)
	if err != nil {
		return err
	}
	if isClosed {
		return fmt.Errorf("connect %v write chan is closed", f.connId)
	}

	//write to chan
	f.writeChan <- data
	return nil
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

//write process
func (f *Connector) writeProcess() {
	var (
		data []byte
		isOk bool
	)
	//defer opt
	defer func() {
		close(f.writeChan)
	}()

	//loop
	for {
		select {
		case data, isOk = <- f.writeChan:
			{
				if isOk && data != nil {
					f.writePureData(data)
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
		m any = nil
	)
	defer func() {
		if pErr := recover(); pErr != m {
			log.Printf("connect %v read process panic, err:%v\n", f.connId, pErr)
		}
		//close message chan
		close(f.messageChan)
	}()

	//start async message worker
	go f.asyncMessageWorker()

	//read process loop
	for {
		if !f.isConnected() {
			continue
		}
		data, err := f.Read(f.conf.MessageType)
		if err != nil {
			if err == io.EOF {
				if f.conf.CBForClosed != nil {
					f.conf.CBForClosed(f.GetConnId())
				}
				break
			}
			if netErr, sok := err.(net.Error); sok && netErr.Timeout() {
				//read timeout, but continue
				continue
			}
			//other error, still continue
			continue
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