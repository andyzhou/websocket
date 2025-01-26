package iface

import (
	"github.com/andyzhou/websocket/gvar"
	"golang.org/x/net/websocket"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * interface of dynamic group
 */
type IGroup interface {
	//gen opt
	Quit()
	Cast(data *gvar.MsgData) error

	//for connect
	CloseConn(connId int64) error
	GetConn(connId int64) (IConnector, error)
	AddConn(connId int64, conn *websocket.Conn) error
}

