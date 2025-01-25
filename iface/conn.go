package iface

import "golang.org/x/net/websocket"

//interface of websocket connect
type IConnector interface {
	//gen opt
	Close()
	CloseWithMessage(message string) error
	GetEntrustGroup() int32
	Entrust(groupId int32, isCancel ...bool) error

	//read and write
	Write(data []byte) error
	Read() ([]byte, error)

	//connect
	GetConnId() int64
	GetConn() *websocket.Conn

	//property
	//GetProperty(key string) (interface{}, error)
	//SetProperty(key string, val interface{}) error
}
