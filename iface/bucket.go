package iface

import (
	"github.com/andyzhou/websocket/gvar"
	"golang.org/x/net/websocket"
)

//interface of bucket
type IBucket interface {
	//gen opt
	Quit()
	Broadcast(data *gvar.MsgData) error

	//for connect
	CloseConn(connId int64) error
	EntrustConn(connId int64, groupId int32, isCancel ...bool) error
	GetConn(connId int64) (IConnector, error)
	AddConn(connId int64, conn *websocket.Conn) error
}
