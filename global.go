package websocket

import "github.com/andyzhou/websocket/gvar"

//get message type
func GetMsgTypeOfJson() int {
	return gvar.MessageTypeOfJson
}
func GetMsgTypeOfOctet() int {
	return gvar.MessageTypeOfOctet
}