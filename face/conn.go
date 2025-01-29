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
	connId       int64           //origin conn id
	conn         *websocket.Conn //origin conn reference
	readTimeout  time.Duration
	writeTimeout time.Duration
}

//construct
//timeouts=> readTimeout, writeTimeout
func NewConnector(
	connId int64,
	conn *websocket.Conn,
	timeouts ...time.Duration) *Connector {
	this := &Connector{
		connId: connId,
		conn: conn,
	}
	this.interInit(timeouts...)
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
	f.conn.SetWriteDeadline(time.Now().Add(f.writeTimeout))

	//send real data
	err := websocket.Message.Send(f.conn, data)
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
	f.conn.SetReadDeadline(time.Now().Add(f.readTimeout))

	//receive data
	err := websocket.Message.Receive(f.conn, &data)
	return data, err
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
		f.readTimeout = time.Second
	}
	if f.writeTimeout <= 0 {
		f.writeTimeout = time.Second
	}
}