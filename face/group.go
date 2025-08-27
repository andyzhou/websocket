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
 * dynamic group face
 * - one group id, one group obj
 * - all connect in one group
 */

//face info
type Group struct {
	groupId        int64
	conf           *gvar.GroupConf //group config reference
	connMap        map[int64]iface.IConnector //connId -> IConnector
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
	f.Lock()
	defer f.Unlock()
	return len(f.connMap)
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

	//remove conn from map
	f.Lock()
	defer func() {
		f.Unlock()

		//check and call the closed cb of outside
		if f.conf != nil && f.conf.CBForClosed != nil {
			f.conf.CBForClosed(f, f.groupId, connId)
		}
	}()
	delete(f.connMap, connId)

	if needRebuildNewMap || len(f.connMap) <= 0 {
		f.rebuild()
	}
	return nil
}

//get connector by owner id
func (f *Group) GetConnByOwnerId(ownerId int64) (iface.IConnector, error) {
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
		connector, ok := v.(iface.IConnector)
		if ok && connector != nil {
			if connector.GetOwnerId() == ownerId {
				targetConnector = connector
				break
			}
		}
	}
	return targetConnector, nil
}

//get old connect
func (f *Group) GetConn(connId int64) (iface.IConnector, error) {
	//check
	if connId <= 0 {
		return nil, errors.New("invalid parameter")
	}

	//get by connect id
	f.Lock()
	defer f.Unlock()
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

	//init new connector
	connector := NewConnector(connId, conn, timeouts...)

	//spawn son process to read and write
	go f.oneConnReadLoop(connector)

	//sync into bucket map with locker
	f.Lock()
	defer func() {
		f.Unlock()
		//check and call the connected cb of outside
		if f.conf != nil && f.conf.CBForConnected != nil {
			f.conf.CBForConnected(f, f.groupId, connId)
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

		f.Lock()
		defer f.Unlock()
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
						connector.Write(data.Data, f.conf.MessageType)
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
				connector.Write(data.Data, f.conf.MessageType)
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