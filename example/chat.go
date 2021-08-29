package main

import (
	"fmt"
	"github.com/andyzhou/websocket/example/json"
	"github.com/andyzhou/websocket/iface"
	"log"
	"sync"
)

/*
 * user router of chat, demo for channel apply.
 *
 * - need apply the interface of IUserRouter
 */

//inter macro define
const (
	//for request kind
	RequestKindOfLogin = "login"
	RequestKindOfChat = "chat"

	//others
	Frequency = 5 //seconds
)

//chat conn info
type ChatConn struct {
	UserId int64
	Nick string
}

//face info
type Chat struct {
	parentRouter iface.IRouter
	users map[int64]*ChatConn //connId -> chatConn
	sync.RWMutex
}

//construct
func NewChat() *Chat {
	//self init
	this := &Chat{
		users:make(map[int64]*ChatConn),
	}
	return this
}

///////////////////////////
//implement of IUserRouter
///////////////////////////

func (f *Chat) Quit() {
	f.users = make(map[int64]*ChatConn)
}

//connect closed
func (f *Chat) OnClose(connId int64) bool {
	log.Println("Chat:OnClose, connId:", connId)
	f.Lock()
	defer f.Unlock()
	delete(f.users, connId)
	return true
}

//hit frequency limit
func (f *Chat) OnFrequencyLimit(connId int64) bool {
	if connId <= 0 {
		return false
	}
	//check user
	v, ok := f.users[connId]
	if !ok {
		return false
	}
	//tips
	tips := fmt.Sprintf("hi, %s, your send too fast!", v.Nick)
	chatOptJson := f.genChatJson("sys", tips)

	//send to current conn
	f.castToConnIds(chatOptJson.Encode(), connId)
	return true
}

//receiver data from client side
func (f *Chat) OnReceiver(connId int64, data interface{}) bool {
	var (
		targetOwnerIds []int64
	)
	log.Printf("Chat:OnReceiver, connId:%v, data:%v\n", connId, data)
	//detect data
	v, ok := data.(map[string]interface{})
	if !ok {
		return false
	}

	//decode gen opt data
	genOptJson := json.NewGenOptJson()
	genOptByte := genOptJson.EncodeSimple(v)
	bRet := genOptJson.Decode(genOptByte)
	if !bRet {
		return false
	}

	//check sub json obj
	subJsonObj, ok := genOptJson.JsonObj.(map[string]interface{})
	if !ok {
		return false
	}

	//init chat opt json
	chatOptJson := json.NewGenOptJson()

	//do relate opt by request kind
	switch genOptJson.Kind {
	case RequestKindOfLogin:
		{
			chatLoginJson := json.NewChatLoginJson()
			loginJsonByte := chatLoginJson.EncodeSimple(subJsonObj)
			bRet = chatLoginJson.Decode(loginJsonByte)
			if !bRet {
				return false
			}

			//login opt
			v, ok := f.users[connId]
			if ok {
				v.UserId = chatLoginJson.Id
				v.Nick = chatLoginJson.Nick
			}

			//get channel and set conn owner
			channel := f.parentRouter.GetChannel()
			if channel != nil {
				conn := channel.GetConnById(connId)
				if conn != nil {
					conn.SetOwnerId(chatLoginJson.Id)
				}
			}

			//tips
			tips := fmt.Sprintf("welcome %s", chatLoginJson.Nick)
			chatOptJson = f.genChatJson("sys", tips)
		}
	case RequestKindOfChat:
		{
			chatInfoJson := json.NewChatInfoJson()
			chatJsonByte := chatInfoJson.EncodeSimple(subJsonObj)
			bRet = chatInfoJson.Decode(chatJsonByte)
			if !bRet {
				return false
			}

			//check login
			v, ok := f.users[connId]
			if !ok {
				return false
			}

			targetOwnerIds = chatInfoJson.AtUserIds
			chatOptJson = f.genChatJson(v.Nick, chatInfoJson.Message)
		}
	}

	//send message
	f.sendData(chatOptJson.Encode(), targetOwnerIds...)

	return true
}

//frame ticker, this works when frame rate > 0
func (f *Chat) OnTick(now int64) {
}

//client connected
func (f *Chat) OnConnect(connId int64) bool {
	fmt.Println("Chat:OnConnect, connId:", connId)
	f.users[connId] = &ChatConn{}
	return true
}

//get frame rate
func (f *Chat) GetFrameRate() int {
	return 0
}

//get frequency
func (f *Chat) GetFrequency() int {
	return Frequency
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

//cast to assigned connect ids
func (f *Chat) castToConnIds(data []byte, connIds ... int64) bool {
	//basic check
	if f.parentRouter == nil || data == nil || connIds == nil {
		return false
	}

	//get channel instance
	channel := f.parentRouter.GetChannel()
	if channel == nil {
		return false
	}

	//cast to channel
	bRet := channel.SendData(data, connIds...)

	return bRet
}

//cast chat data
func (f *Chat) sendData(data []byte, targetIds ...int64) bool {
	var (
		connIds []int64
	)

	//basic check
	if f.parentRouter == nil || data == nil {
		return false
	}

	//get channel instance
	channel := f.parentRouter.GetChannel()
	if channel == nil {
		return false
	}

	//get connect id for target user
	connIds = make([]int64, 0)
	if targetIds != nil {
		for _, targetId := range targetIds {
			for connId, v := range f.users {
				if v.UserId == targetId {
					connIds = append(connIds, connId)
				}
			}
		}
	}

	//cast to channel
	bRet := channel.SendData(data, connIds...)

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