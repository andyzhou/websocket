package gvar

import "time"

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * global variables define
 */

type (
	//persistent router conf
	RouterConf struct {
		//general
		Uri          string
		Buckets      int
		ReadTimeout  time.Duration
		WriteTimeout time.Duration

		//cb func for websocket
		CBForGenConnId func() int64
		CBForConnected func(router interface{}, connId int64) error
		CBForClosed    func(router interface{}, connId int64) error
		CBForRead      func(router interface{}, connId int64, message []byte) error
	}

	//dynamic group conf
	GroupConf struct {
		//general
		Uri          string //final format like `/<orgUri>/{groupId}`
		ReadTimeout  time.Duration
		WriteTimeout time.Duration

		//cb func for websocket
		CBForGenConnId   func() int64
		CBForVerifyGroup func(groupObj interface{}, groupId int64) error
		CBForConnected   func(groupObj interface{}, groupId int64, connId int64) error
		CBForClosed      func(groupObj interface{}, groupId int64, connId int64) error
		CBForRead        func(groupObj interface{}, groupId int64, connId int64, message []byte) error
	}

	MsgData struct {
		Data    []byte
		ConnIds []int64
	}
)