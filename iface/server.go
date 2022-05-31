package iface

import (
	"golang.org/x/net/websocket"
	"net/http"
)

/*
 * interface of web socket server
 */

type IWServer interface {
	Quit()
	Start()

	//get
	GetWsParas(ws *websocket.Conn) map[string]string
	GetTotalRouter() int64
	GetRouter(channel string) IRouter
	GetWSServer() IWServer

	//set
	SetStaticPath(staticPath string)
	SetTplPath(tplPath string)
	SetGlobalTplFile(tplFile ... string) bool
	SetTplAutoLoad(auto bool)

	//register router
	RegisterStaticRouter() bool
	RegisterPageRouter(
			subUrl string,
			cb func(r *http.Request,
		) interface{}) bool
	RegisterHttpRouter(
			subUrl string,
			cb func(w http.ResponseWriter, r *http.Request),
			method ... string,
		) bool
	RegisterChannelRouter(
			channel string,
			userRouter IUserRouter,
		) bool
}
