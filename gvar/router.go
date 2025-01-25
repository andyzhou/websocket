package gvar

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * global variables define
 */

type (
	RouterConf struct {
		//general
		Uri     string
		Buckets int

		//relate cb func
		CBForGenConnId func() int64
		CBForConnected func(routerUri string, connId int64) error
		CBForClosed    func(routerUri string, connId int64) error
		CBForRead      func(routerUri string, connId int64, message []byte) error
	}

	GroupConf struct {
		GroupId        int32 //assigned group id
		CBForConnected func(groupId int32, connId int64) error
		CBForClosed    func(groupId int32, connId int64) error
		CBForRead      func(groupId int32, connId int64, message []byte) error
	}

	MsgData struct {
		Data    []byte
		ConnIds []int64
		GroupId int32
	}
)