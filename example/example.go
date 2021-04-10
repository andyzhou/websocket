package main

import (
	"fmt"
	"github.com/andyzhou/websocket"
	"github.com/andyzhou/websocket/define"
	"net/http"
	"sync"
)

const (
	serverPort = 7200
	reqUrlOfRoot = "/"
	reqUrlOfFile = "/file/{file}"
	reqUrlOfChat = "/chat/{channel}"
	channel = "test"
	staticPath = "html"
)


func main() {
	var wg sync.WaitGroup

	//init web socket server
	server := websocket.NewServer(serverPort)

	//get inter ws
	ws := server.GetWSServer()

	//set static root path
	ws.SetStaticPath(staticPath)

	//register handler for http router
	ws.RegisterHttpRouter(reqUrlOfRoot, rootHandler, define.RouterMethodOfGet)

	//register handler for static router
	ws.RegisterStaticRouter(reqUrlOfFile)

	//register handler for web socket router
	//chat := NewChat()
	//ws.RegisterWSRouter(reqUrlOfChat, channel, chat)

	//start server
	wg.Add(1)
	fmt.Println("start server..")
	server.Start()

	wg.Wait()
	fmt.Println("stop server..")
	server.Quit()
}


//root handler
func rootHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}


}