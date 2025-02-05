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
func cbForClosed(group interface{}, groupId int64, connId int64) error {
	groupObj, _ := group.(iface.IGroup)
	if groupObj == nil {
		return errors.New("invalid group obj")
	}
	log.Printf("example.cbForClosed, groupId:%v, connId:%v\n", groupId, connId)
	return nil
}

//cb for connected
func cbForConnected(group interface{}, groupId int64, connId int64) error {
	groupObj, _ := group.(iface.IGroup)
	if groupObj == nil {
		return errors.New("invalid group obj")
	}
	log.Printf("example.cbForConnected, groupId:%v, connId:%v\n", groupId, connId)
	return nil
}

//cb for verify group
func cbForVerifyGroup(group interface{}, groupId int64) error {
	log.Printf("example.cbForVerifyGroup, groupId:%v\n", groupId)
	return nil
}

//cb for read data from client sent
func cbForReadData(group interface{}, groupId int64, connId int64, data []byte) error {
	log.Printf("example.cbForReadData, groupId:%v, connId:%v, data:%v\n", groupId, connId, string(data))

	groupObj, _ := group.(iface.IGroup)
	if groupObj == nil {
		return errors.New("invalid group obj")
	}

	//format msg data
	msgData := s.GenMsgData()
	msgData.Data = data

	//cast to all
	groupObj.Cast(msgData)
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
