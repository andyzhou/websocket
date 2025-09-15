package face

import (
	"errors"
	"fmt"
	"github.com/andyzhou/websocket/define"
	"time"

	"github.com/andyzhou/websocket/gvar"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * websocket connect face
 */

//face info
type Connector struct {
	connId       int64            //origin conn id
	ownerId      int64
	activeTime   int64
	conn         *websocket.Conn //origin conn reference
	writeChan    chan []byte     //write byte chan
	closeChan    chan bool
	readTimeout  time.Duration
	writeTimeout time.Duration
	Util
}

//construct
//timeouts=> readTimeout, writeTimeout
func NewConnector(
	connId int64,
	conn *websocket.Conn,
	timeouts ...time.Duration) *Connector {
	this := &Connector{
		connId:    connId,
		conn:      conn,
		writeChan: make(chan []byte, define.ConnWriteChanSize),
		closeChan: make(chan bool, 1),
	}
	this.interInit(timeouts...)
	return this
}

//close
func (f *Connector) Close() {
	if f.conn != nil {
		close(f.closeChan)
		f.conn.Close()
		f.conn = nil
	}
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
	f.conn.Write([]byte(message))
	f.conn.Close()
	f.conn = nil

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

func (f *Connector) GetUriQueryPara(keyName string) string {
	if f.conn == nil {
		return ""
	}
	return f.conn.Request().URL.Query().Get(keyName)
}

//get origin connect reference
func (f *Connector) GetConn() *websocket.Conn {
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
	f.conn.SetWriteDeadline(time.Now().Add(f.writeTimeout))

	//send real data
	switch messageType {
	case gvar.MessageTypeOfJson:
		{
			//json format
			err = websocket.JSON.Send(f.conn, data)
		}
	case gvar.MessageTypeOfOctet:
		fallthrough
	default:
		{
			//general octet format
			err = websocket.Message.Send(f.conn, data)
		}
	}
	return err
}

//read message with timeout
func (f *Connector) Read(messageTypes ...int) (interface{}, error) {
	var (
		messageType int
		err error
	)
	//check
	if f.conn == nil {
		return nil, errors.New("connect is nil")
	}
	if messageTypes != nil && len(messageTypes) > 0 {
		messageType = messageTypes[0]
	}

	//set read deadline
	if f.conn == nil {
		return nil, errors.New("connect is nil")
	}
	f.conn.SetReadDeadline(time.Now().Add(f.readTimeout))

	//update active time
	defer func() {
		f.updateActiveTime(time.Now().Unix())
	}()

	//receive data
	switch messageType {
	case gvar.MessageTypeOfJson:
		{
			//json format
			var data interface{}
			err = websocket.JSON.Receive(f.conn, &data)
			return data, err
		}
	case gvar.MessageTypeOfOctet:
		fallthrough
	default:
		{
			//general octet format
			var data []byte
			err = websocket.Message.Receive(f.conn, &data)
			return data, err
		}
	}
}

//update active time
func (f *Connector) updateActiveTime(ts int64) {
	f.activeTime = ts
}

//pop queue to write
func (f *Connector) queuePopProcess() {
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
					f.conn.Write(data)
				}
			}
		case <- f.closeChan:
			{
				return
			}
		}
	}
}

//inter init
func (f *Connector) interInit(timeouts ...time.Duration) {
	//setup timeouts
	timeoutsLen := len(timeouts)
	switch timeoutsLen {
	case 1:
		{
			f.readTimeout = timeouts[0]
		}
	case 2:
		{
			f.readTimeout = timeouts[0]
			f.writeTimeout = timeouts[1]
		}
	}
	if f.readTimeout <= 0 {
		f.readTimeout = time.Second/10
	}
	if f.writeTimeout <= 0 {
		f.writeTimeout = time.Second/10
	}

	//update active time
	f.updateActiveTime(time.Now().Unix())

	//run queue pop process
	go f.queuePopProcess()
}