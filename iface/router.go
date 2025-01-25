package iface

import (
	"github.com/andyzhou/websocket/gvar"
	"golang.org/x/net/websocket"
)

//interface of router
type IRouter interface {
	//gen opt
	Quit()
	GetConf() *gvar.RouterConf
	Entry(conn *websocket.Conn)
}
