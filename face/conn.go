package face

import (
	"errors"
	"golang.org/x/net/websocket"
	"time"
)

/*
 * face of web socket conn, implement of IConn
 */

//face info
type Conn struct {
	connId int64
	ownerId int64
	blockEndTime int64
	conn *websocket.Conn
}

//construct
func NewConn(connId int64, conn *websocket.Conn) *Conn {
	//self init
	this := &Conn{
		connId: connId,
		conn: conn,
	}
	return this
}

//conn close
func (f *Conn) Close() {
	if f.conn != nil {
		f.conn.Close()
	}
}

//get conn owner id
func (f *Conn) GetOwnerId() int64 {
	return f.ownerId
}

//get conn id
func (f *Conn) GetConnId() int64 {
	return f.connId
}

//block conn
func (f *Conn) Block(endTime int64) {
	f.blockEndTime = endTime
}

//set conn owner id
func (f *Conn) SetOwnerId(ownerId int64) {
	f.ownerId = ownerId
}

//send data
func (f *Conn) SendData(data []byte) (err error) {
	//basic check
	if data == nil || f.conn == nil {
		err = errors.New("invalid parameter")
		return
	}
	if f.blockEndTime > 0 {
		now := time.Now().Unix()
		if f.blockEndTime > now {
			//still blocked
			err = errors.New("conn be blocked")
			return
		}
		f.blockEndTime = 0
	}

	//try catch panic
	defer func() {
		if subErr := recover(); subErr != nil {
			err = errors.New(err.Error())
			return
		}
	}()

	//write data to conn
	_, err = f.conn.Write(data)
	return
}