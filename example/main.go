package main

import (
	"fmt"
	"github.com/andyzhou/websocket"
	"net/http"
	"sync"
)

const (
	serverPort = 7200
	staticPath = "html"
	tplPath = "tpl"
	reqSubPageOfTest = "test"
	reqSubReqOfTest = "test"
	reqSubChannelOfTest = "test"
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

	//register handler for static router
	ws.RegisterStaticRouter()

	//register handler for page router
	ws.RegisterPageRouter(reqSubPageOfTest, cbForTestPage)

	//register handler for http request router
	ws.RegisterHttpRouter(reqSubReqOfTest, cbForTestReqResp)

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

//cb for test request response
func cbForTestReqResp(w http.ResponseWriter, r *http.Request) {
	fmt.Println("cbForTestReqResp")
	w.Write([]byte("hi"))
}

//cb for test page tpl data
//need return hash map
func cbForTestPage(r *http.Request) interface{} {
	fmt.Println("cbForTestPage")
	//get key parameter of query
	//params := mux.Vars(r)
	//name, _ := params["name"]
	name := r.URL.Query().Get("name")
	fmt.Println("para of name:", name)

	//fill tpl data
	data := make(map[string]interface{})
	data["Title"] = "Hello"

	return data
}

//root handler
func rootHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
}