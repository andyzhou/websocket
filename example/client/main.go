package main

import (
	"fmt"
	"time"

	wsClient "github.com/andyzhou/websocket"
)

const (
	WsHost 	   = "ws://localhost"
	WsPort     = 8080
)

//send message loop
func sendMessage(client *wsClient.Client) {
	var (
		err error
	)
	for {
		err = client.SendText("test")
		if err != nil {
			break
		}
		time.Sleep(time.Second * 3)
	}
}

func main() {
	wsAddr := fmt.Sprintf("%v:%v/group/1", WsHost, WsPort)
	client := wsClient.NewClient(
		wsAddr,
		"http://localhost/",
	)

	client.OnConnect = func() {
		fmt.Println("Connected!")
		client.SendText("Hello WebSocket")
		go sendMessage(client)
	}

	client.OnMessage = func(mt wsClient.MessageType, data []byte) {
		fmt.Println("Received:", string(data))
	}

	client.OnError = func(err error) {
		fmt.Println("Error:", err)
	}

	client.OnClose = func() {
		fmt.Println("Closed")
	}

	err := client.Connect()
	if err != nil {
		panic(any(err))
	}

	time.Sleep(30 * time.Second)
	client.Close()
}