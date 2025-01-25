package iface

import (
	"github.com/andyzhou/websocket/gvar"
	"golang.org/x/net/websocket"
)

//interface of router
type IRouter interface {
	//gen opt
	Quit()
	GetConf() *gvar.RouterConf
	Broadcast(msg *gvar.MsgData) error
	Entry(conn *websocket.Conn)

	//for dynamic group
	DelGroup(groupId int32) error
	GetGroup(groupId int32) (IGroup, error)
	JoinGroup(groupId int32, connId int64, isCancel ...bool) error
	CreateGroup(conf *gvar.GroupConf) (IGroup, error)
}
