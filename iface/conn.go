package iface

import "golang.org/x/net/websocket"

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * interface of connector
 */
type IConnector interface {
	//gen opt
	Close()
	CloseWithMessage(message string) error
	GetUriParas() map[string]string

	//read and write
	Write(data []byte) error
	Read() ([]byte, error)

	//connect
	GetConnId() int64
	GetConn() *websocket.Conn
}
