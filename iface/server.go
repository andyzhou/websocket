package iface

import (
	"golang.org/x/net/websocket"
	"net/http"
)

/*
 * interface of web socket server
 */

type IServer interface {
	Quit()
	Start()
	GetWsParas(ws *websocket.Conn) map[string]string
	GetNewConnId() int64
	GetTotalRouter() int64
	GetRouter(channel string) IRouter
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
