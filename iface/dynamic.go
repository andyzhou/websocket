package iface

import (
	"github.com/andyzhou/websocket/gvar"
	"golang.org/x/net/websocket"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * interface of dynamic router
 */

type IDynamic interface {
	Quit()
	GetConf() *gvar.GroupConf
	GetGroup(groupId int32) (IGroup, error)
	Cast(groupId int32, msg *gvar.MsgData) error
	Entry(conn *websocket.Conn)
}