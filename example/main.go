package main

import (
	"github.com/andyzhou/websocket"
	"sync"
)

/*
 * example code
 */

const (
	WsUri     = "/ws"
	WsBuckets = 3
	WsPort    = 8080
)

//cb for closed
func cbForClosed(uri string, connId int64) error {
	return nil
}

//cb for connected
func cbForConnected(uri string, connId int64) error {
	return nil
}

func main() {
	var (
		wg sync.WaitGroup
	)

	//init server
	s := websocket.NewServer()

	//setup config
	routerCfg := s.GenRouterCfg()
	routerCfg.Uri = WsUri
	routerCfg.Buckets = WsBuckets

	//setup cb opt
	routerCfg.CBForConnected = cbForConnected
	routerCfg.CBForClosed = cbForClosed

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
