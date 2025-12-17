package main

import (
	"errors"
	"log"
	"sync"

	genWebsocket "golang.org/x/net/websocket"

	"github.com/andyzhou/websocket"
	"github.com/andyzhou/websocket/gvar"
	"github.com/andyzhou/websocket/iface"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * example code for server side
 */

/*
 * if use json format, client js send data like these:
 `
	var messageObj = new Object();
	messageObj.messge = message;
  	ws.send(JSON.stringify(messageObj));
 `
 */

const (
	WsGroupUri = "/group"
	WsPort     = 8080
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
	//get origin connect and path para
	conn, _ := groupObj.GetConn(connId)
	pathParas := conn.GetUriParas()
	queryParas := conn.GetUriQueryParas()
	nameVal := queryParas.Get("name")
	ageVal := queryParas.Get("age")

	log.Printf("example.cbForConnected, groupId:%v, connId:%v, pathParas:%v, nameVal:%v, ageVal:%v\n",
		groupId, connId, pathParas, nameVal, ageVal)
	return nil
}

//cb for verify group
func cbForVerifyGroup(conn *genWebsocket.Conn, group interface{}, groupId int64) error {
	log.Printf("example.cbForVerifyGroup, groupId:%v\n", groupId)
	return nil
}

//cb for read data from client sent
func cbForReadData(group interface{}, groupId int64, connId int64, messageType int, data interface{}) error {
	var (
		msgData *gvar.MsgData
	)
	if data == nil {
		return errors.New("invalid parameter")
	}
	log.Printf("example.cbForReadData, groupId:%v, connId:%v, messageType:%v, data:%v\n",
		groupId, connId, messageType, data)
	groupObj, _ := group.(iface.IGroup)
	if groupObj == nil {
		return errors.New("invalid group obj")
	}

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
				msgData.Data = []byte(string(byteData))
			}
			break
		}
	}
	//cast to all
	msgData.WriteInQueue = true
	err := groupObj.Cast(msgData)
	return err
}

func main() {
	var (
		wg sync.WaitGroup
	)

	//init server
	s = websocket.NewServer()

	//setup config
	routerCfg := s.GenGroupCfg()
	routerCfg.Uri = WsGroupUri
	routerCfg.MessageType = gvar.MessageTypeOfOctet

	//setup cb opt
	routerCfg.CBForVerifyGroup = cbForVerifyGroup
	routerCfg.CBForConnected = cbForConnected
	routerCfg.CBForClosed = cbForClosed
	routerCfg.CBForRead = cbForReadData

	//register dynamic router
	dynamic, err := s.RegisterDynamic(routerCfg)
	if err != nil {
		panic(any(err))
	}

	//create group
	group, subErr := dynamic.CreateGroup(1)
	if subErr != nil {
		panic(any(subErr))
	}
	groupId := group.GetId()
	connects := group.GetTotal()
	log.Printf("group:%v, conns:%v\n", groupId, connects)

	//start service
	wg.Add(1)
	err = s.Start(WsPort)
	if err != nil {
		wg.Done()
		panic(any(err))
	}

	wg.Wait()
}
