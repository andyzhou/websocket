package face

import (
	"github.com/andyzhou/websocket/define"
	"github.com/andyzhou/websocket/iface"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
	"log"
	"sync/atomic"
)

/*
 * face of router, implement of IRouter
 */

//face info
type Router struct {
	tag string
	channel iface.IChannel
	userRouter iface.IUserRouter
	connId int64
}

//construct
func NewRouter(tag string, userRouter iface.IUserRouter) *Router {
	//self init
	this := &Router{
		tag: tag,
		channel: NewChannel(tag),
		userRouter: userRouter,
	}
	return this
}

//quit
func (f *Router) Quit()  {
	if f.channel != nil {
		f.channel.Quit()
	}
}

//get channel instance
func (f *Router) GetChannel() iface.IChannel {
	return f.channel
}

//main entry for router
func (f *Router) Entry(conn *websocket.Conn) {
	var (
		err error
		genVal interface{}
	)

	//basic check
	if conn == nil {
		return
	}

	//defer
	defer func() {
		if subErr := recover(); subErr != nil {
			log.Println("Router:Entry panic, err:", subErr)
		}
	}()

	//check channel name
	params := mux.Vars(conn.Request())
	if params == nil {
		return
	}
	channelTag, ok := params[define.ChannelParaName]
	if !ok || channelTag != f.tag {
		return
	}

	//gen new connect id
	newConnId := f.genConnId()

	//init IConn
	connClient := NewConn(newConnId, conn)

	//init web socket receiver
	socketReceiver := websocket.JSON

	//add connect into channel
	f.channel.NewConn(connClient)

	//loop
	for {
		//receive data from client
		err = socketReceiver.Receive(conn, genVal)
		if err != nil {
			log.Println("Router:Entry err:", err.Error())
			break
		}
		//call cb for received data
		if f.userRouter != nil {
			f.userRouter.OnReceiver(genVal)
		}
	}
}

/////////////////
//private func
/////////////////

//gen new conn id
func (f *Router) genConnId() int64 {
	return atomic.AddInt64(&f.connId, 1)
}