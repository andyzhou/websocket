package face

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/andyzhou/websocket/define"
	"github.com/andyzhou/websocket/gvar"
	"github.com/andyzhou/websocket/iface"
	"golang.org/x/net/websocket"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * dynamic group for dynamic face
 * - one group id, one group obj
 * - all connects in one group
 */

//face info
type Group struct {
	groupId        int64
	conf           *gvar.GroupConf //group config reference
	connMap        map[int64]iface.IConnector //connId -> IConnector
	connOwnerMap   map[int64]int64 //ownerId -> connId
	writeChan      chan gvar.MsgData
	writeCloseChan chan bool
	sync.RWMutex
	Util
}

//construct
func NewGroup(groupId int64, cfg *gvar.GroupConf) *Group {
	this := &Group{
		groupId:        groupId,
		conf:           cfg,
		connMap:        map[int64]iface.IConnector{},
		connOwnerMap:   map[int64]int64{},
		writeChan:      make(chan gvar.MsgData, define.DefaultGroupWriteChan),
		writeCloseChan: make(chan bool, 1),
	}
	this.interInit()
	return this
}

//quit
func (f *Group) Quit() {
	//force close main loop
	close(f.writeCloseChan)

	//just del record
	f.Lock()
	defer f.Unlock()
	for _, v := range f.connMap {
		if v != nil {
			v.Close()
		}
	}

	//release old map
	f.connMap = nil
}

//broadcast to connections by condition
func (f *Group) Cast(data *gvar.MsgData) error {
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
		return fmt.Errorf("group %v write chan had closed", f.groupId)
	}

	//send to write chan
	f.writeChan <- *data
	return nil
}

//get group id
func (f *Group) GetId() int64 {
	return f.groupId
}

//get total connects
func (f *Group) GetTotal() int {
	f.RLock()
	defer f.RUnlock()
	return len(f.connMap)
}

//set conn owner id
func (f *Group) SetOwner(connId, ownerId int64) error {
	//check
	if connId <= 0 || ownerId <= 0 {
		return errors.New("invalid parameter")
	}

	//get connector
	connector, _ := f.GetConn(connId)
	if connector == nil {
		return errors.New("can't get connector by id")
	}
	connector.SetOwnerId(ownerId)

	f.Lock()
	defer f.Unlock()
	f.connOwnerMap[ownerId] = connId
	return nil
}

//close old connect
func (f *Group) CloseConn(connId int64) error {
	//check
	if connId <= 0 {
		return errors.New("invalid parameter")
	}

	//hit gc rate
	gcRate := rand.Intn(define.FullPercent)
	needRebuildNewMap := false
	if gcRate > 0 && gcRate <= define.DynamicGroupGcRate {
		needRebuildNewMap = true
	}

	//check and call the closed cb of outside
	if f.conf != nil && f.conf.CBForClosed != nil {
		f.conf.CBForClosed(f, f.groupId, connId)
	}

	//get conn obj
	connector, _ := f.GetConn(connId)
	if connector != nil {
		connector.Close()
	}

	//remove conn from map
	f.Lock()
	delete(f.connMap, connId)
	if connector != nil {
		delete(f.connOwnerMap, connector.GetOwnerId())
	}
	f.Unlock()

	if needRebuildNewMap || len(f.connMap) <= 0 {
		f.rebuild()
	}
	return nil
}

//get connector by owner id
func (f *Group) GetConnByOwnerId(ownerId int64) (iface.IConnector, error) {
	//check
	if ownerId <= 0 {
		return nil, errors.New("invalid parameter")
	}

	//loop map to found
	f.RLock()
	defer f.RUnlock()
	connId, ok := f.connOwnerMap[ownerId]
	if !ok || connId <= 0 {
		return nil, errors.New("can't get owner id")
	}

	//get conn by id
	connector, sok := f.connMap[connId]
	if !sok || connector == nil {
		return nil, errors.New("can't get connector by id")
	}
	return connector, nil
}

//get old connect
func (f *Group) GetConn(connId int64) (iface.IConnector, error) {
	//check
	if connId <= 0 {
		return nil, errors.New("invalid parameter")
	}

	//get by connect id
	f.RLock()
	defer f.RUnlock()
	v, ok := f.connMap[connId]
	if !ok || v == nil {
		return nil, errors.New("no such connector")
	}
	return v, nil
}

//add new connect
func (f *Group) AddConn(connId int64, conn *websocket.Conn, timeouts ...time.Duration) error {
	//check
	if connId <= 0 || conn == nil {
		return errors.New("invalid parameter")
	}

	//setup connect config
	cbForRead := func(connId int64, messageType int, data interface{}) error {
		if f.conf.CBForRead != nil {
			return f.conf.CBForRead(f, f.groupId, connId, messageType, data)
		}
		return nil
	}
	cbForClose := func(connId int64) error {
		return f.CloseConn(connId)
	}
	connConf := &ConnConf{
		MessageType: f.conf.MessageType,
		CBForRead: cbForRead,
		CBForClosed: cbForClose,
	}

	//init new connector
	connector := NewConnector(connConf, connId, conn, timeouts...)

	//sync into bucket map with locker
	f.Lock()
	defer func() {
		f.Unlock()
		//check and call the connected cb of outside
		if f.conf != nil && f.conf.CBForConnected != nil {
			f.conf.CBForConnected(f, f.groupId, connector)
		}
	}()
	f.connMap[connId] = connector
	return nil
}

////////////////
//private func
////////////////

//rebuild inter map
func (f *Group) rebuild() {
	//release old conn map
	f.Lock()
	defer f.Unlock()
	newConnMap := map[int64]iface.IConnector{}
	for k, v := range f.connMap {
		newConnMap[k] = v
	}
	f.connMap = newConnMap
}

//one conn read loop
//fix multi concurrency read issue
func (f *Group) oneConnReadLoop(conn iface.IConnector) {
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
			log.Printf("group %v oneConnReadLoop panic, err:%v\n", f.groupId, pErr)
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
				f.conf.CBForRead(f, f.groupId, conn.GetConnId(), f.conf.MessageType, data)
			}
		}
	}
}

//write loop
func (f *Group) writeLoop() {
	var (
		msgData gvar.MsgData
		m any = nil
	)
	//panic catch
	defer func() {
		if pErr := recover(); pErr != m {
			log.Printf("group %v wriet loop panic, err:%v\n", f.groupId, pErr)
		}
		if f.writeChan != nil {
			f.writeChan = nil
		}
	}()

	//sub func for write opt
	subWriteFunc := func(data *gvar.MsgData) error {
		//check
		if data == nil || data.Data == nil {
			return errors.New("invalid parameter")
		}

		//check byte data
		byteData, _ := data.Data.([]byte)

		f.Lock()
		defer f.Unlock()

		//send by owner ids
		if data.OwnerIds != nil && len(data.OwnerIds) > 0 {
			for _, ownerId := range data.OwnerIds {
				if ownerId <= 0 {
					continue
				}
				connId, ok := f.connOwnerMap[ownerId]
				if !ok || connId <= 0 {
					continue
				}
				v, ok := f.connMap[connId]
				if ok && v != nil {
					connector, sok := v.(iface.IConnector)
					if sok && connector != nil {
						//write to target conn
						if data.WriteInQueue {
							connector.QueueWrite(byteData)
						}else{
							connector.Write(data.Data, f.conf.MessageType)
						}
					}
				}
			}
		}

		//send by conn ids
		if data.ConnIds != nil && len(data.ConnIds) > 0 {
			//send to assigned connect ids
			for _, connId := range data.ConnIds {
				if connId <= 0 {
					continue
				}
				v, ok := f.connMap[connId]
				if ok && v != nil {
					connector, sok := v.(iface.IConnector)
					if sok && connector != nil {
						//write to target conn
						if data.WriteInQueue {
							connector.QueueWrite(byteData)
						}else{
							connector.Write(data.Data, f.conf.MessageType)
						}
					}
				}
			}
			return nil
		}

		//send to all connects
		for _, v := range f.connMap {
			connector, sok := v.(iface.IConnector)
			if sok && connector != nil {
				//write to target conn
				if data.WriteInQueue {
					connector.QueueWrite(byteData)
				}else{
					connector.Write(data.Data, f.conf.MessageType)
				}
			}
		}
		return nil
	}

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
				subWriteFunc(&msgData)
			}
		}
	}
}

//remove connect
func (f *Group) removeConnect(connId int64) {
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

//inter init
func (f *Group) interInit() {
	//spawn main write loop
	go f.writeLoop()
}