package iface

import (
	"time"

	"github.com/andyzhou/websocket/gvar"
	"golang.org/x/net/websocket"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * interface of bucket
 */
type IBucket interface {
	//gen opt
	Quit()
	Broadcast(data *gvar.MsgData) error
	SetOwner(connId, ownerId int64) error

	//for connect
	CloseConn(connId int64) error
	GetConnByOwnerId(ownerId int64) (IConnector, error)
	GetConn(connId int64) (IConnector, error)
	AddConn(connId int64, conn *websocket.Conn, timeouts ...time.Duration) error
}
