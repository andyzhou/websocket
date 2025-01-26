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
	Cast(msg *gvar.MsgData) error
	Entry(conn *websocket.Conn)

	////for dynamic group
	//DelGroup(groupId int32) error
	//GetGroup(groupId int32) (IGroup, error)
	//JoinGroup(groupId int32, connId int64, isCancel ...bool) error
	//CreateGroup(conf *gvar.GroupConf) (IGroup, error)
}
