package gvar

import "time"

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * global variables define
 */

//message type
const (
	MessageTypeOfOctet = iota //BinaryMessage
	MessageTypeOfJson         //JsonMessage
)

type (
	//persistent router conf
	RouterConf struct {
		//general
		Uri          string
		Buckets      int
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		MessageType  int

		//cb func for websocket
		CBForGenConnId func() int64
		CBForConnected func(router interface{}, connId int64) error
		CBForClosed    func(router interface{}, connId int64) error
		CBForRead      func(router interface{}, connId int64, messageType int, data interface{}) error
	}

	//dynamic group conf
	GroupConf struct {
		//general
		Uri          string //final format like `/<orgUri>/{groupId}`
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		MessageType  int

		//cb func for websocket
		CBForGenConnId   func() int64
		CBForVerifyGroup func(groupObj interface{}, groupId int64) error
		CBForConnected   func(groupObj interface{}, groupId int64, connId int64) error
		CBForClosed      func(groupObj interface{}, groupId int64, connId int64) error
		CBForRead        func(groupObj interface{}, groupId int64, connId int64, messageType int, data interface{}) error
	}

	MsgData struct {
		Data       interface{}
		ConnIds    []int64
	}
)