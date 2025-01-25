package gvar

/*
 * global variables define
 */

type (
	RouterConf struct {
		//general
		Uri              string
		//Method 			 string //`GET` or `POST`
		//MsgType          int
		Buckets          int
		//HeartByte        []byte
		//MaxActiveSeconds int

		//relate cb func
		CBForGenConnId func() int64
		CBForConnected func(routerUri string, connId int64) error
		CBForClosed    func(routerUri string, connId int64) error
		CBForRead      func(routerUri string, connId int64, message []byte) error
	}
)