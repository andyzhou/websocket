package iface

/*
 * interface of web socket conn
 */

type IConn interface {
	Close()
	GetOwnerId() int64
	GetConnId() int64
	Block(endTime int64)
	SetOwnerId(ownerId int64)
	SendData(data []byte) error
}
