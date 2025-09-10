package iface

import (
	"time"

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
	GetTotal() int
	GetId() int64

	//for connect
	SetOwner(connId, ownerId int64) error
	RemoveConn(connId int64) error
	CloseConn(connId int64) error
	GetConnByOwnerId(ownerId int64) (IConnector, error)
	GetConn(connId int64) (IConnector, error)
	AddConn(connId int64, conn *websocket.Conn, timeouts ...time.Duration) error
}

