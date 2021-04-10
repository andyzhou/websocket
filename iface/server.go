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
	GetNewConnId() int64
	GetTotalRouter() int64
	GetRouter(channel string) IRouter

	//set
	SetStaticPath(staticPath string)

	//register router
	RegisterStaticRouter(subUrl string) bool
	RegisterHttpRouter(
			subUrl string,
			cb func(w http.ResponseWriter, r *http.Request),
			method ... string,
		) bool
	RegisterWSRouter(
			subUrl,
			channel string,
			userRouter IUserRouter,
		) bool
}
