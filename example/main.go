package main

import (
	"fmt"
	"github.com/andyzhou/websocket"
	"net/http"
	"sync"
)

const (
	serverPort = 7200
	reqUrlOfRoot = "/{page}"
	reqUrlOfFile = "/file/{file}"
	reqUrlOfChat = "/chat/{channel}"
	channel = "test"
	staticPath = "html"
	tplPath = "tpl"
)

//global tpl files
var globalTplFiles = []string {
	"header.tpl",
}

func main() {
	var wg sync.WaitGroup

	//init web socket server
	server := websocket.NewServer(serverPort)

	//get inter ws
	ws := server.GetWSServer()

	//set static root path
	ws.SetStaticPath(staticPath)

	//set tpl root path
	ws.SetTplPath(tplPath)

	//set global tpl files
	ws.SetGlobalTplFile(globalTplFiles...)

	//register handler for page router
	ws.RegisterPageRouter(reqUrlOfRoot)

	//register handler for static router
	ws.RegisterStaticRouter(reqUrlOfFile)

	//register handler for web socket router
	//chat := NewChat()
	//ws.RegisterWSRouter(reqUrlOfChat, channel, chat)

	//start server
	wg.Add(1)
	fmt.Println("start example server..")
	server.Start()

	wg.Wait()
	fmt.Println("stop example server..")
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