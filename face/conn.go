package face

import (
	"errors"
	"github.com/gorilla/mux"
	"time"

	"golang.org/x/net/websocket"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * websocket connect face
 */

//face info
type Connector struct {
	connId         int64           //origin conn id
	conn           *websocket.Conn //origin conn reference
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

//get uri paras
func (f *Connector) GetUriParas() map[string]string {
	if f.conn == nil {
		return nil
	}
	return mux.Vars(f.conn.Request())
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