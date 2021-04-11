package main

import (
	"fmt"
	"github.com/andyzhou/websocket/iface"
)

/*
 * user router of chat
 *
 * - need apply the interface of IUserRouter
 */

type Chat struct {
	parentRouter iface.IRouter
}

//construct
func NewChat() *Chat {
	//self init
	this := &Chat{
	}
	return this
}

///////////////////////////
//implement of IUserRouter
///////////////////////////

func (f *Chat) Quit() {
}

func (f *Chat) OnClose(connId int64) bool {
	fmt.Print("Chat:OnClose, connId:", connId)
	return true
}

func (f *Chat) OnReceiver(data interface{}) bool {
	fmt.Print("Chat:OnReceiver, data:", data)
	return true
}

func (f *Chat) OnConnect(connId int64) bool {
	fmt.Print("Chat:OnConnect, connId:", connId)
	return true
}

func (f *Chat) GetChannelParaName() string {
	return ""
}

func (f *Chat) SetParentRouter(router iface.IRouter) bool {
	if router == nil {
		return false
	}
	f.parentRouter = router
	return true
}