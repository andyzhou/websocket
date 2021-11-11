package face

import (
	"github.com/andyzhou/websocket/iface"
	"log"
	"sync"
	"sync/atomic"
)

/*
 * face of web socket channel, implement of IChannel
 */

//inter macro define
const (
	channelSendChanSize = 1024 * 5
)

//send queue
type sendQueue struct {
	data []byte
	connIds []int64 //if not assigned, will send to all.
}

//face info
type Channel struct {
	tag string
	connMap *sync.Map //connId -> IConn
	connCount int64
	sendChan chan sendQueue
	closeChan chan bool
}

//face info
func NewChannel(tag string) *Channel {
	//self init
	this := &Channel{
		tag: tag,
		connMap: new(sync.Map),
		sendChan: make(chan sendQueue, channelSendChanSize),
		closeChan: make(chan bool, 1),
	}
	//spawn main process
	go this.runMainProcess()
	return this
}

//quit
func (f *Channel) Quit() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Channel:Quit panic, err:", err)
		}
	}()

	//cleanup conn map
	cf := func(k, v interface{}) bool {
		conn, ok := v.(iface.IConn)
		if !ok {
			return false
		}
		conn.Close()
		return true
	}
	f.connMap.Range(cf)

	//send to close chan
	f.closeChan <- true
}

//get total connect count
func (f *Channel) GetTotalConn() int64 {
	return f.connCount
}

//get IConn by id
func (f *Channel) GetConnById(connId int64) iface.IConn {
	//basic check
	if connId <= 0 || f.connMap == nil || f.connCount <= 0 {
		return nil
	}
	return f.getConnById(connId)
}

//add connect
func (f *Channel) NewConn(conn iface.IConn) bool {
	//basic check
	if conn == nil {
		return false
	}

	//check old
	connId := conn.GetConnId()
	old := f.getConnById(connId)
	if old != nil {
		return true
	}

	//add new
	f.connMap.Store(connId, conn)
	atomic.AddInt64(&f.connCount, 1)

	return true
}

//remove conn by id
func (f *Channel) RemoveConn(connId int64) bool {
	if connId <= 0 {
		return false
	}
	f.connMap.Delete(connId)
	atomic.AddInt64(&f.connCount, -1)
	return true
}

//block conn
func (f *Channel) BlockConn(connId, endTime int64) bool {
	if connId <= 0 {
		return false
	}
	conn := f.getConnById(connId)
	if conn == nil {
		return false
	}
	conn.Block(endTime)
	return true
}

//send data
func (f *Channel) SendData(data []byte, connIds ... int64) (bRet bool) {
	//basic check
	if data == nil || f.connCount <= 0 {
		bRet = false
		return
	}

	//try catch panic
	defer func() {
		if err := recover(); err != nil {
			log.Println("Channel:SendData panic, err:", err)
			bRet = false
		}
	}()

	//init queue data
	queue := sendQueue{
		data: data,
		connIds: connIds,
	}

	//async send to chan
	select {
	case f.sendChan <- queue:
	}
	bRet = true
	return
}

///////////////
//private func
///////////////

//run main process
func (f *Channel) runMainProcess() {
	var (
		queue sendQueue
		isOk bool
	)

	//defer
	defer func() {
		if err := recover(); err != nil {
			log.Println("Channel:runMainProcess panic, err:", err)
		}
		//close chan
		close(f.sendChan)
		close(f.closeChan)
	}()

	//async loop
	for {
		select {
		case queue, isOk = <- f.sendChan:
			if isOk {
				f.sendData(&queue)
			}
		case <- f.closeChan:
			return
		}
	}
}

//send data
func (f *Channel) sendData(queue *sendQueue) bool {
	if queue == nil {
		return false
	}
	bRet := false
	if queue.connIds == nil || len(queue.connIds) <= 0 {
		//send to all
		bRet = f.sendDataToAll(queue.data)
	}else{
		//send to assigned conn ids
		bRet = f.sendDataToConnIds(queue.connIds, queue.data)
	}
	return bRet
}

//send to assigned conn ids
func (f *Channel) sendDataToConnIds(connIds []int64, data []byte) bool {
	//basic check
	if connIds == nil || len(connIds) <= 0 {
		return false
	}
	for _, connId := range connIds {
		conn := f.getConnById(connId)
		if conn == nil {
			continue
		}
		conn.SendData(data)
	}
	return true
}

//send to all
func (f *Channel) sendDataToAll(data []byte) bool {
	sf := func(k, v interface{}) bool {
		conn, ok := v.(iface.IConn)
		if !ok {
			return false
		}
		conn.SendData(data)
		return true
	}
	f.connMap.Range(sf)
	return true
}

//get conn by id
func (f *Channel) getConnById(connId int64) iface.IConn {
	//basic check
	if connId <= 0 || f.connCount <= 0 {
		return nil
	}
	//get IConn by id
	v, ok := f.connMap.Load(connId)
	if !ok {
		return nil
	}
	conn, ok := v.(iface.IConn)
	if !ok {
		return nil
	}
	return conn
}