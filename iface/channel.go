package iface

/*
 * interface of web socket channel
 */

type IChannel interface {
	Quit()
	SendData(data []byte, connId ... int64) bool
	RemoveConn(connId int64) bool
	BlockConn(connId, endTime int64) bool
	GetTotalConn() int64
	GetConnById(connId int64) IConn
	NewConn(conn IConn) bool
}