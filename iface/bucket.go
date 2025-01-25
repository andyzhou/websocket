package iface

import "golang.org/x/net/websocket"

//interface of bucket
type IBucket interface {
	Quit()
	CloseConn(connId int64) error
	GetConn(connId int64) (IConnector, error)
	AddConn(connId int64, conn *websocket.Conn) error
}
