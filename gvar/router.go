package gvar

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * global variables define
 */

type (
	//persistent router conf
	RouterConf struct {
		//general
		Uri     string
		Buckets int

		//cb func for websocket
		CBForGenConnId func() int64
		CBForConnected func(routerUri string, connId int64) error
		CBForClosed    func(routerUri string, connId int64) error
		CBForRead      func(routerUri string, connId int64, message []byte) error
	}

	//dynamic group conf
	GroupConf struct {
		//general
		Uri              string //final format like `/<orgUri>/{groupId}`

		//cb func for websocket
		CBForGenConnId   func() int64
		CBForVerifyGroup func(routerUri string, groupId int32) error
		CBForConnected   func(routerUri string, groupId int32, connId int64) error
		CBForClosed      func(routerUri string, groupId int32, connId int64) error
		CBForRead        func(routerUri string, groupId int32, connId int64, message []byte) error
	}

	MsgData struct {
		Data    []byte
		ConnIds []int64
	}
)