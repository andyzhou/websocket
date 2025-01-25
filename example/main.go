package main

import (
	"github.com/andyzhou/websocket"
	"log"
	"sync"
)

/*
 * example code for server side
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
func cbForClosed(uri string, connId int64) error {
	log.Printf("example.cbForClosed, connId:%v\n", connId)
	return nil
}

//cb for connected
func cbForConnected(uri string, connId int64) error {
	log.Printf("example.cbForConnected, connId:%v\n", connId)
	return nil
}

//cb for read data from client sent
func cbForReadData(uri string, connId int64, data []byte) error {
	log.Printf("example.cbForReadData, connId:%v, data:%v\n", connId, string(data))
	//cast to all
	if s != nil {
		subRouter, _ := s.GetRouter(WsUri)
		if subRouter != nil {
			//format msg data
			msgData := s.GenMsgData()
			msgData.Data = data
			subRouter.Broadcast(msgData)
		}
	}
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
