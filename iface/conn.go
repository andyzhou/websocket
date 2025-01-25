package iface

import "golang.org/x/net/websocket"

//interface of websocket connect
type IConnector interface {
	//gen opt
	Close()
	//CloseWithMessage(message string) error

	//read and send
	//SendData(data []byte) error
	Read() ([]byte, error)

	//get
	GetConnId() int64
	GetConn() *websocket.Conn
	//GetProperty(key string) (interface{}, error)

	//set
	//SetProperty(key string, val interface{}) error
}
