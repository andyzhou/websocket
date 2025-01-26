package main

import (
	"github.com/andyzhou/websocket"
	"log"
	"sync"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * example code for server side
 */

const (
	WsUri     = "/group"
	WsBuckets = 3
	WsPort    = 8080
)

//global variable
var (
	s *websocket.Server
)

//cb for closed
func cbForClosed(uri string, groupId int32, connId int64) error {
	log.Printf("example.cbForClosed, uri:%v, groupId:%v, connId:%v\n",
			uri, groupId, connId)
	return nil
}

//cb for connected
func cbForConnected(uri string, groupId int32, connId int64) error {
	log.Printf("example.cbForConnected, uri:%v, groupId:%v, connId:%v\n",
		uri, groupId, connId)
	return nil
}

//cb for verify group
func cbForVerifyGroup(uri string, groupId int32) error {
	log.Printf("example.cbForVerifyGroup, uri:%v, groupId:%v\n", uri, groupId)
	return nil
}

//cb for read data from client sent
func cbForReadData(uri string, groupId int32, connId int64, data []byte) error {
	log.Printf("example.cbForReadData, uri:%v, groupId:%v, connId:%v, data:%v\n",
				uri, groupId, connId, string(data))
	//cast to all
	if s != nil {
		subDynamic, _ := s.GetDynamic(uri)
		if subDynamic != nil {
			//format msg data
			msgData := s.GenMsgData()
			msgData.Data = data
			subDynamic.Cast(groupId, msgData)
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
	routerCfg := s.GenGroupCfg()
	routerCfg.Uri = WsUri

	//setup cb opt
	routerCfg.CBForVerifyGroup = cbForVerifyGroup
	routerCfg.CBForConnected = cbForConnected
	routerCfg.CBForClosed = cbForClosed
	routerCfg.CBForRead = cbForReadData

	//register dynamic router
	err := s.RegisterDynamic(routerCfg)
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
