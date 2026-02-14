package websocket

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * websocket client face
 */

type MessageType int

const (
	TextMessage MessageType = iota
	BinaryMessage
)

type writeMessage struct {
	messageType MessageType
	data        []byte
}

//client option
type ClientOption struct {
	Reconnect            bool
	ReconnectMax         int
	ReconnectCount       int
	ReconnectBaseSeconds int
	HeartbeatSeconds     int
}

//client face
type Client struct {
	url    string
	origin string

	conn   *websocket.Conn
	connMu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	writeChan chan writeMessage

	reconnect       bool
	reconnectMax    int
	reconnectCount  int
	reconnectBase   time.Duration
	heartbeatPeriod time.Duration

	closeOnce sync.Once

	//cb functions
	OnConnect func()
	OnMessage func(messageType MessageType, data []byte)
	OnError   func(err error)
	OnClose   func()
}

//new client
func NewClient(url, origin string, options ...ClientOption) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		url:             url,
		origin:          origin,
		ctx:             ctx,
		cancel:          cancel,
		writeChan:       make(chan writeMessage, 1024),
		reconnect:       true,
		reconnectMax:    10,
		reconnectBase:   2 * time.Second,
		heartbeatPeriod: 30 * time.Second,
	}
	if len(options) > 0 {
		option := options[0]
		client.reconnect = option.Reconnect
		client.reconnectMax = option.ReconnectMax
		client.reconnectBase = time.Duration(option.ReconnectBaseSeconds) * time.Second
		client.heartbeatPeriod = time.Duration(option.HeartbeatSeconds) * time.Second
	}
	return client
}

//connect server
func (c *Client) Connect() error {
	config, err := websocket.NewConfig(c.url, c.origin)
	if err != nil {
		return err
	}

	conn, subErr := websocket.DialConfig(config)
	if subErr != nil {
		return subErr
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	c.reconnectCount = 0

	if c.OnConnect != nil {
		c.OnConnect()
	}

	go c.readLoop()
	go c.writeLoop()
	go c.heartbeatLoop()

	return nil
}

//read loop
func (c *Client) readLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			var msg []byte
			err := websocket.Message.Receive(c.conn, &msg)
			if err != nil {
				c.handleError(err)
				return
			}

			if c.OnMessage != nil {
				c.OnMessage(TextMessage, msg)
			}
		}
	}
}

//write loop
func (c *Client) writeLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.writeChan:
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			if conn == nil {
				continue
			}

			var err error
			if msg.messageType == TextMessage {
				err = websocket.Message.Send(conn, string(msg.data))
			} else {
				err = websocket.Message.Send(conn, msg.data)
			}

			if err != nil {
				c.handleError(err)
				return
			}
		}
	}
}

//heart beat
func (c *Client) heartbeatLoop() {
	ticker := time.NewTicker(c.heartbeatPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.SendText("ping")
		}
	}
}


//send text data
func (c *Client) SendText(data string) error {
	select {
	case c.writeChan <- writeMessage{
		messageType: TextMessage,
		data:        []byte(data),
	}:
		return nil
	case <-c.ctx.Done():
		return errors.New("client closed")
	}
}

func (c *Client) SendBinary(data []byte) error {
	select {
	case c.writeChan <- writeMessage{
		messageType: BinaryMessage,
		data:        data,
	}:
		return nil
	case <-c.ctx.Done():
		return errors.New("client closed")
	}
}

//handle error and reconnect
func (c *Client) handleError(err error) {
	if c.OnError != nil {
		c.OnError(err)
	}

	if !c.reconnect {
		c.Close()
		return
	}

	c.reconnectCount++
	if c.reconnectCount > c.reconnectMax {
		log.Println("max reconnect reached")
		c.Close()
		return
	}

	backoff := time.Duration(c.reconnectCount) * c.reconnectBase
	time.Sleep(backoff)

	log.Println("reconnecting...")

	if err := c.Connect(); err != nil {
		c.handleError(err)
	}
}


//close connect
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.cancel()

		c.connMu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.connMu.Unlock()

		close(c.writeChan)

		if c.OnClose != nil {
			c.OnClose()
		}
	})
}
