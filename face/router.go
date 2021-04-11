package face

import (
	"github.com/andyzhou/websocket/define"
	"github.com/andyzhou/websocket/iface"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
	"log"
	"reflect"
	"sync/atomic"
	"time"
)

/*
 * face of web socket router, implement of IRouter
 */

//inter macro define
const (
	routerFrameRateMax = 60 //max frame rate
)

//face info
type Router struct {
	tag string
	channel iface.IChannel
	userRouter iface.IUserRouter
	connId int64
	frameRate int
	tickCloseChan chan bool
}

//construct
func NewRouter(tag string, userRouter iface.IUserRouter) *Router {
	//self init
	this := &Router{
		tag: tag,
		channel: NewChannel(tag),
		userRouter: userRouter,
		tickCloseChan: make(chan bool, 1),
	}

	//check user router frame rate
	this.frameRate = userRouter.GetFrameRate()
	if this.frameRate > routerFrameRateMax {
		this.frameRate = routerFrameRateMax
	}

	//spawn inter process
	go this.runTickerProcess()
	return this
}

//quit
func (f *Router) Quit()  {
	//defer
	defer func() {
		if err := recover(); err != nil {
			log.Println("Router:Quit panic, err:", err)
		}
	}()

	if f.channel != nil {
		f.channel.Quit()
	}

	//close chan
	f.tickCloseChan <- true
}

//get channel instance
func (f *Router) GetChannel() iface.IChannel {
	return f.channel
}

//main entry for router
func (f *Router) Entry(conn *websocket.Conn) {
	var (
		connId int64
		err error
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
		//call `OnClose` of IUserRouter
		if f.userRouter != nil {
			f.userRouter.OnClose(connId)
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
	connId = f.genConnId()

	//init IConn
	connClient := NewConn(connId, conn)

	//call `OnConnect` of IUserRouter
	if f.userRouter != nil {
		f.userRouter.OnConnect(connId)
	}

	//init web socket receiver
	socketReceiver := websocket.JSON

	//add connect into channel
	f.channel.NewConn(connClient)

	//init general value
	genVal := make(map[string]interface{})

	//loop
	for {
		//receive data from client
		err = socketReceiver.Receive(conn, &genVal)
		if err != nil {
			log.Println("Router:Entry err:", err.Error())
			break
		}
		//call cb for received data
		if f.userRouter != nil && genVal != nil {
			f.userRouter.OnReceiver(genVal)
		}
	}
}

/////////////////
//private func
/////////////////

//inter ticker process
func (f *Router) runTickerProcess() {
	//check frame rate
	if f.frameRate <= 0 {
		return
	}

	//init ticker
	duration := time.Duration(1/f.frameRate) * time.Second
	ticker := time.NewTicker(duration)

	//defer
	defer func() {
		if err := recover(); err != nil {
			log.Println("Router:runTickerProcess panic, err:", err)
		}
		ticker.Stop()
		close(f.tickCloseChan)
	}()

	//loop
	for {
		select {
		case <- ticker.C:
			{
				if f.userRouter != nil {
					f.userRouter.OnTick(time.Now().Unix())
				}
			}
		}
	}
}

//reset json object
func (f *Router) resetJsonObject(v interface{}) {
	p := reflect.ValueOf(v).Elem()
	p.Set(reflect.Zero(p.Type()))
}

//gen new conn id
func (f *Router) genConnId() int64 {
	return atomic.AddInt64(&f.connId, 1)
}