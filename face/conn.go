package face

import (
	"errors"
	"time"

	"golang.org/x/net/websocket"
)

/*
 * websocket connect face
 */

//face info
type Connector struct {
	connId         int64           //origin conn id
	conn           *websocket.Conn //origin conn reference
	entrustGroupId int32           //entrusted by group id
}

//construct
func NewConnector(connId int64, conn *websocket.Conn) *Connector {
	this := &Connector{
		connId: connId,
		conn: conn,
	}
	return this
}

//close
func (f *Connector) Close() {
	if f.conn != nil {
		f.conn.Close()
		f.conn = nil
	}
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

//get entrusted group id
func (f *Connector) GetEntrustGroup() int32 {
	return f.entrustGroupId
}

//entrust or not group id
func (f *Connector) Entrust(groupId int32, isCancel ...bool) error {
	var (
		cancelOpt bool
	)
	//check
	if groupId <= 0 {
		return errors.New("invalid parameter")
	}
	//check cancel opt
	if isCancel != nil && len(isCancel) > 0 {
		cancelOpt = isCancel[0]
	}

	if cancelOpt {
		f.entrustGroupId = 0
	}else{
		f.entrustGroupId = groupId
	}
	return nil
}

//get origin connect reference
func (f *Connector) GetConn() *websocket.Conn {
	return f.conn
}

//send message with timeout
func (f *Connector) Write(data []byte) error {
	//check
	if data == nil {
		return errors.New("invalid parameter")
	}
	if f.conn == nil {
		return errors.New("connect is nil")
	}

	//set write deadline
	f.conn.SetWriteDeadline(time.Now().Add(time.Second))

	//send real data
	err := websocket.Message.Send(f.conn, string(data))
	return err
}

//read message with timeout
func (f *Connector) Read() ([]byte, error) {
	var (
		data []byte
	)
	//check
	if f.conn == nil {
		return nil, errors.New("connect is nil")
	}

	//set read deadline
	f.conn.SetReadDeadline(time.Now().Add(time.Second))

	//receive data
	err := websocket.Message.Receive(f.conn, &data)
	return data, err
}