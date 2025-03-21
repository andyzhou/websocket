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
	GetUriQueryPara(keyName string) string
	UpdateActiveTime(ts int64)
	GetActiveTime() int64

	//owner id
	GetOwnerId() int64
	SetOwnerId(ownerId int64)

	//read and write
	Write(data interface{}, messageTypes ...int) error
	Read(messageTypes ...int) (interface{}, error)

	//connect
	GetConnId() int64
	GetConn() *websocket.Conn
}
