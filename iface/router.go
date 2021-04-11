package iface

import "golang.org/x/net/websocket"

/*
 * interface of web socket router
 */

//user side router
//need apply relate interface func
type IUserRouter interface {
	Quit()
	GetChannelParaName() string
	SetParentRouter(router IRouter) bool
	OnReceiver(data interface{}) bool
	OnClose(connId int64) bool
	OnConnect(connId int64) bool
}

//inter router
type IRouter interface {
	Quit()
	GetChannel() IChannel
	Entry(conn *websocket.Conn)
}
