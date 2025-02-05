package main

import (
	"errors"
	"github.com/andyzhou/websocket"
	"github.com/andyzhou/websocket/iface"
	"log"
	"sync"
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
func cbForClosed(router interface{}, connId int64) error {
	routerObj, _ := router.(iface.IRouter)
	if routerObj == nil {
		return errors.New("invalid router")
	}
	log.Printf("example.cbForClosed, connId:%v\n", connId)
	return nil
}

//cb for connected
func cbForConnected(router interface{}, connId int64) error {
	routerObj, _ := router.(iface.IRouter)
	if routerObj == nil {
		return errors.New("invalid router")
	}
	log.Printf("example.cbForConnected, connId:%v\n", connId)
	return nil
}

//cb for read data from client sent
func cbForReadData(router interface{}, connId int64, data []byte) error {
	routerObj, _ := router.(iface.IRouter)
	if routerObj == nil {
		return errors.New("invalid router")
	}
	log.Printf("example.cbForReadData, connId:%v, data:%v\n", connId, string(data))
	//format msg data
	msgData := s.GenMsgData()
	msgData.Data = data

	//cast to all
	routerObj.Cast(msgData)
	return nil
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
