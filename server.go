package websocket

import (
	"github.com/andyzhou/websocket/face"
	"github.com/andyzhou/websocket/iface"
)

/*
 * interface for server
 */

//face info
type Server struct {
	ws iface.IWServer //web socket server
}

//construct
func NewServer(port int) *Server {
	//self init
	this := &Server{
		ws: face.NewWServer(port),
	}
	return this
}

//quit
func (s *Server) Quit() {
	s.ws.Quit()
}

//start
func (s *Server) Start() {
	s.ws.Start()
}

//get web socket server
func (s *Server) GetWSServer() iface.IWServer {
	return s.ws
}
