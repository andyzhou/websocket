package main

import (
	"fmt"
	"github.com/andyzhou/websocket/example/json"
	"github.com/andyzhou/websocket/iface"
)

/*
 * user router of chat
 *
 * - need apply the interface of IUserRouter
 */

//chat json
type ChatJson struct {
	Kind string `json:"kind"`
}

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


//connect closed
func (f *Chat) OnClose(connId int64) bool {
	fmt.Println("Chat:OnClose, connId:", connId)
	return true
}

//receiver data from client side
func (f *Chat) OnReceiver(data interface{}) bool {
	fmt.Println("Chat:OnReceiver, data:", data)

	//init chat json
	genOptJson := f.genChatJson("sys", "chat test")

	//send message
	f.castToAll(genOptJson.Encode())

	return true
}

//frame ticker
func (f *Chat) OnTick(now int64) {
}

//client connected
func (f *Chat) OnConnect(connId int64) bool {
	fmt.Println("Chat:OnConnect, connId:", connId)
	return true
}

//get frame rate
func (f *Chat) GetFrameRate() int {
	return 0
}

//set parent router
func (f *Chat) SetParentRouter(router iface.IRouter) bool {
	if router == nil {
		return false
	}
	f.parentRouter = router
	return true
}

///////////////
//private func
///////////////

//cast to all
func (f *Chat) castToAll(data []byte) bool {
	//basic check
	if f.parentRouter == nil || data == nil {
		return false
	}

	//get channel instance
	channel := f.parentRouter.GetChannel()
	if channel == nil {
		return false
	}

	//cast to channel
	bRet := channel.SendData(data)

	return bRet
}

//gen chat json
func (f *Chat) genChatJson(nick, message string) *json.GenOptJson {
	//chat json
	chatJson := json.NewChatJson()
	chatJson.SenderNick = nick
	chatJson.Message = message

	//gen opt json
	genOptJson := json.NewGenOptJson()
	genOptJson.Kind = "chat"
	genOptJson.JsonObj = chatJson

	return genOptJson
}