package face

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andyzhou/websocket/define"
	"github.com/andyzhou/websocket/gvar"
	"github.com/andyzhou/websocket/iface"
	"golang.org/x/net/websocket"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * sub bucket face for router
 * - one bucket service batch connections
 */

//face info
type Bucket struct {
	bucketId       int
	router         iface.IRouter              //reference
	conf           *gvar.RouterConf           //reference of parent router
	connMap        map[int64]iface.IConnector //connId -> IConnector
	writeChan      chan gvar.MsgData
	writeCloseChan chan bool
	opts           int64
	sync.RWMutex
	Util
}

//construct
func NewBucket(router iface.IRouter, bucketId int, cfg *gvar.RouterConf) *Bucket {
	this := &Bucket{
		router: router,
		bucketId: bucketId,
		conf: cfg,
		connMap: map[int64]iface.IConnector{},
		writeChan: make(chan gvar.MsgData, define.DefaultBucketWriteChan),
		writeCloseChan: make(chan bool, 1),
	}
	this.interInit()
	go this.periodicCheck()
	return this
}

//quit
func (f *Bucket) Quit() {
	if f.writeCloseChan != nil {
		//force close main loop
		close(f.writeCloseChan)
	}

	//force close with locker
	f.Lock()
	defer f.Unlock()
	for _, v := range f.connMap {
		if v != nil {
			v.Close()
		}
	}
	f.connMap = nil

	//release old map
	atomic.StoreInt64(&f.opts, 0)
	newConnMap := map[int64]iface.IConnector{}
	f.connMap = newConnMap
}

//broadcast to connections by condition
func (f *Bucket) Broadcast(data *gvar.MsgData) error {
	//check
	if data == nil || data.Data == nil {
		return errors.New("invalid parameter")
	}

	//check chan is closed or not
	chanIsClosed, err := f.IsChanClosed(f.writeChan)
	if err != nil {
		return err
	}
	if chanIsClosed {
		return fmt.Errorf("bucket %v write chan had closed", f.bucketId)
	}

	//send to write chan
	f.writeChan <- *data
	return nil
}

//close old connect
func (f *Bucket) CloseConn(connId int64) error {
	//check
	if connId <= 0 {
		return errors.New("invalid parameter")
	}

	//get conn by id
	connector, _ := f.GetConn(connId)
	if connector != nil {
		//force close connect
		connector.Close()
	}

	//remove conn from map
	f.Lock()
	delete(f.connMap, connId)
	f.Unlock()

	//atomic opt
	if f.opts > 0 {
		atomic.AddInt64(&f.opts, 1)
	}

	//check and call the closed cb of outside
	if f.conf != nil && f.conf.CBForClosed != nil {
		f.conf.CBForClosed(f.router, connId)
	}

	//hit gc rate
	gcRate := rand.Intn(define.FullPercent)
	needCopyNewMap := false
	if gcRate > 0 && gcRate <= define.DynamicGroupGcRate {
		needCopyNewMap = true
	}

	//release old map
	if needCopyNewMap || len(f.connMap) <= 0 {
		f.rebuild()
	}
	return nil
}

//get connector by owner id
func (f *Bucket) GetConnByOwnerId(ownerId int64) (iface.IConnector, error) {
	var (
		targetConnector iface.IConnector
	)
	//check
	if ownerId <= 0 {
		return nil, errors.New("invalid parameter")
	}

	//loop map to found
	f.Lock()
	defer f.Unlock()
	for _, v := range f.connMap {
		if v != nil && v.GetOwnerId() == ownerId {
			targetConnector = v
			break
		}
	}
	return targetConnector, nil
}

//get old connect
func (f *Bucket) GetConn(connId int64) (iface.IConnector, error) {
	//check
	if connId <= 0 {
		return nil, errors.New("invalid parameter")
	}

	//get connect by id
	f.Lock()
	defer f.Unlock()
	conn, ok := f.connMap[connId]
	if !ok || conn == nil {
		return nil, errors.New("no such connector")
	}
	return conn, nil
}

//add new connect
func (f *Bucket) AddConn(connId int64, conn *websocket.Conn, timeouts ...time.Duration) error {
	//check
	if connId <= 0 || conn == nil {
		return errors.New("invalid parameter")
	}

	//init new connector
	connector := NewConnector(connId, conn, timeouts...)

	//spawn son process to read
	go f.oneConnReadLoop(connector)

	//sync into bucket map with locker
	f.Lock()
	f.connMap[connId] = connector
	f.Unlock()

	//atomic opt
	atomic.AddInt64(&f.opts, 1)

	//check and call the connected cb of outside
	if f.conf != nil && f.conf.CBForConnected != nil {
		f.conf.CBForConnected(f.router, connId)
	}
	return nil
}

////////////////
//private func
////////////////

//one conn read loop
//fix multi concurrency read issue
func (f *Bucket) oneConnReadLoop(conn iface.IConnector) {
	var (
		data interface{}
		m any = nil
		err error
	)
	//check
	if conn == nil {
		return
	}

	//defer opt
	defer func() {
		if pErr := recover(); pErr != m {
			log.Printf("bucket %v oneConnReadLoop panic, err:%v\n", f.bucketId, pErr)
		}
	}()

	//read loop
	for {
		//read data
		data, err = conn.Read(f.conf.MessageType)
		if err != nil {
			if err == io.EOF {
				//lost or read a bad connect
				//remove and force close connect
				f.CloseConn(conn.GetConnId())
				break
			}
			if netErr, sok := err.(net.Error); sok && netErr.Timeout() {
				//read timeout
				//do nothing
			}
		}else{
			//read data succeed
			//check and call the read cb of outside
			if f.conf != nil && f.conf.CBForRead != nil {
				f.conf.CBForRead(f.router, conn.GetConnId(), f.conf.MessageType, data)
			}
		}
	}
}

//sub write message opt
func (f *Bucket) subWriteOpt(data *gvar.MsgData) error {
	//check
	if data == nil || data.Data == nil {
		return errors.New("invalid parameter")
	}

	//check byte data
	byteData, _ := data.Data.([]byte)

	f.Lock()
	defer f.Unlock()
	if data.ConnIds != nil && len(data.ConnIds) > 0 {
		//send to assigned connect ids
		for _, connId := range data.ConnIds {
			if connId <= 0 {
				continue
			}
			conn, ok := f.connMap[connId]
			if !ok || conn == nil {
				continue
			}

			//write to target conn
			if data.WriteInQueue {
				conn.QueueWrite(byteData)
			}else{
				conn.Write(data.Data, f.conf.MessageType)
			}
		}
		return nil
	}

	//send to all connects
	for _, conn := range f.connMap {
		//write to target conn
		if data.WriteInQueue {
			conn.QueueWrite(byteData)
		}else{
			conn.Write(data.Data, f.conf.MessageType)
		}
	}
	return nil
}

//write loop
func (f *Bucket) writeLoop() {
	var (
		msgData gvar.MsgData
		m any = nil
	)
	//panic catch
	defer func() {
		if pErr := recover(); pErr != m {
			log.Printf("bucket %v wriet loop panic, err:%v\n", f.bucketId, pErr)
		}
		if f.writeChan != nil {
			f.writeChan = nil
		}
		if f.writeCloseChan != nil {
			f.writeCloseChan = nil
		}
	}()

	//loop opt
	for {
		select {
		case <- f.writeCloseChan:
			{
				//force quit write loop
				return
			}
		case msgData = <- f.writeChan:
			{
				//write inter message data
				f.subWriteOpt(&msgData)
			}
		}
	}
}

//remove connect
func (f *Bucket) removeConnect(connId int64) {
	if connId <= 0 {
		return
	}
	f.Lock()
	defer f.Unlock()
	delete(f.connMap, connId)
	if len(f.connMap) <= 0 {
		newConnMap := map[int64]iface.IConnector{}
		f.connMap = newConnMap
	}
}

//rebuild
func (f *Bucket) rebuild() {
	f.Lock()
	defer f.Unlock()
	newConnMap := map[int64]iface.IConnector{}
	for k, v := range f.connMap {
		newConnMap[k] = v
	}
	f.connMap = newConnMap
	atomic.StoreInt64(&f.opts, 0)

	//force gc opt
	runtime.GC()
}

//dynamic connect map check
func (f *Bucket) periodicCheck() {
	//init ticker
	ticker := time.NewTicker(define.BucketRebuildSeconds * time.Second)
	defer ticker.Stop()

	//loop ticker
	for range ticker.C {
		if f.opts > 0 {
			f.rebuild()
		}
	}
}

//inter init
func (f *Bucket) interInit() {
	go f.writeLoop()
}
