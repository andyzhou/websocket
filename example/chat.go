package main

/*
 * user router of chat
 *
 * - need apply the interface of IUserRouter
 */

type Chat struct {
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
	return true
}

func (f *Chat) OnReceiver(data interface{}) bool {
	return true
}

func (f *Chat) OnConnect(connId int64) bool {
	return true
}

func (f *Chat) GetChannelParaName() string {
	return ""
}