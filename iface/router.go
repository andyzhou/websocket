package iface

import (
	"github.com/andyzhou/websocket/gvar"
	"golang.org/x/net/websocket"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * interface of persistent router
 */
type IRouter interface {
	//gen opt
	Quit()
	GetConf() *gvar.RouterConf
	GetConnector(connId int64) (IConnector, error)
	SwitchBucket(connectId int64, from, to int) error
	Cast(msg *gvar.MsgData) error
	SetOwner(connId, ownerId int64) error
	Entry(conn *websocket.Conn)
}
