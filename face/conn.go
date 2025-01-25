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
	connId int64           //origin conn id
	conn   *websocket.Conn //origin conn reference
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
}

//get origin connect id
func (f *Connector) GetConnId() int64 {
	return f.connId
}

//get origin connect reference
func (f *Connector) GetConn() *websocket.Conn {
	return f.conn
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