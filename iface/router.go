package iface

import "golang.org/x/net/websocket"

/*
 * interface of web socket router
 */

//user side router
//need apply relate interface func
type IUserRouter interface {
	Quit()

	//get
	GetFrameRate() int
	GetFrequency() int

	//set
	SetParentRouter(router IRouter) bool

	//cb
	OnFrequencyLimit(connId int64) bool
	OnTick(now int64)
	OnReceiver(connId int64, data interface{}) bool
	OnClose(connId int64) bool
	OnConnect(connId int64) bool
}

//inter base router
type IRouter interface {
	Quit()
	GetChannel() IChannel
	Entry(conn *websocket.Conn)
}
