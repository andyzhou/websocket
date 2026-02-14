package main

import (
	"errors"
	"log"
	"sync"

	"github.com/andyzhou/websocket"
	"github.com/andyzhou/websocket/gvar"
	"github.com/andyzhou/websocket/iface"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * router example code for server side
 */

const (
	WsUri     = "/ws"
	WsBuckets = 3
	WsPort    = 8080
)

//global variable
var (
	s *websocket.Server
)

//cb for closed
func cbForClosed(router interface{}, bucketId int, connId int64) error {
	routerObj, _ := router.(iface.IRouter)
	if routerObj == nil {
		return errors.New("invalid router")
	}
	log.Printf("example.cbForClosed, bucketId:%v, connId:%v\n", bucketId, connId)
	return nil
}

//cb for connected
func cbForConnected(router interface{}, bucketId int, connector interface{}) error {
	routerObj, _ := router.(iface.IRouter)
	if routerObj == nil {
		return errors.New("invalid router")
	}
	connect, _ := connector.(iface.IConnector)
	if connect == nil {
		return errors.New("invalid connector")
	}

	log.Printf("example.cbForConnected, bucketId:%v, connId:%v\n", bucketId, connect.GetConnId())
	return nil
}

//cb for read data from client sent
func cbForReadData(router interface{}, bucketId int, connId int64, messageType int, data interface{}) error {
	var (
		msgData *gvar.MsgData
	)
	routerObj, _ := router.(iface.IRouter)
	if routerObj == nil {
		return errors.New("invalid router")
	}
	log.Printf("example.cbForReadData, bucketId:%v, connId:%v, messageType:%v, data:%v\n",
		bucketId, connId, messageType, data)

	//init msg data
	msgData = s.GenMsgData()

	//do diff opt by message type
	switch messageType {
	case gvar.MessageTypeOfJson:
		{
			//json format
			msgData.Data = data
			break
		}
	case gvar.MessageTypeOfOctet:
		{
			//string format
			byteData, ok := data.([]uint8)
			if ok && byteData != nil {
				msgData.Data = string(byteData)
			}
			break
		}
	}

	//cast to all
	err := routerObj.Cast(msgData)
	return err
}

func main() {
	var (
		wg sync.WaitGroup
	)

	//init server
	s = websocket.NewServer()

	//setup config
	routerCfg := s.GenRouterCfg()
	routerCfg.Uri = WsUri
	routerCfg.Buckets = WsBuckets
	routerCfg.MessageType = gvar.MessageTypeOfOctet

	//setup cb opt
	routerCfg.CBForConnected = cbForConnected
	routerCfg.CBForClosed = cbForClosed
	routerCfg.CBForRead = cbForReadData

	//register router
	err := s.RegisterRouter(routerCfg)
	if err != nil {
		panic(any(err))
	}

	//start service
	wg.Add(1)
	err = s.Start(WsPort)
	if err != nil {
		panic(any(err))
	}

	wg.Wait()
}
