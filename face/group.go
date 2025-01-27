package face

import (
	"errors"
	"fmt"
	"github.com/andyzhou/websocket/define"
	"github.com/andyzhou/websocket/gvar"
	"github.com/andyzhou/websocket/iface"
	"golang.org/x/net/websocket"
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
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
	groupId        int32
	conf           *gvar.GroupConf //group config reference
	connMap        sync.Map        //connId -> IConnector
	connects       int32           //total connects
	writeChan      chan gvar.MsgData
	readCloseChan  chan bool
	writeCloseChan chan bool
	Util
}

//construct
func NewGroup(groupId int32, cfg *gvar.GroupConf) *Group {
	this := &Group{
		groupId: groupId,
		conf: cfg,
		connMap: sync.Map{},
		writeChan: make(chan gvar.MsgData, define.DefaultGroupWriteChan),
		readCloseChan: make(chan bool, 1),
		writeCloseChan: make(chan bool, 1),
	}
	this.interInit()
	return this
}


//quit
func (f *Group) Quit() {
	//force close main loop
	close(f.writeCloseChan)
	close(f.readCloseChan)

	//just del record
	rangeOpt := func(k, v interface{}) bool {
		f.connMap.Delete(k)
		return true
	}
	f.connMap.Range(rangeOpt)

	//gc opt
	runtime.GC()
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

//close old connect
func (f *Group) CloseConn(connId int64) error {
	//check
	if connId <= 0 {
		return errors.New("invalid parameter")
	}

	//remove conn from map
	f.connMap.Delete(connId)

	//check and call the closed cb of outside
	if f.conf != nil && f.conf.CBForClosed != nil {
		f.conf.CBForClosed(f.conf.Uri, f.groupId, connId)
	}
	atomic.AddInt32(&f.connects, -1)
	if f.connects <= 0 {
		atomic.StoreInt32(&f.connects, 0)
		runtime.GC()
	}
	return nil
}

//get old connect
func (f *Group) GetConn(connId int64) (iface.IConnector, error) {
	//check
	if connId <= 0 {
		return nil, errors.New("invalid parameter")
	}
	v, ok := f.connMap.Load(connId)
	if !ok || v == nil {
		return nil, errors.New("no such connector")
	}

	//detect value
	connector, sok := v.(iface.IConnector)
	if !sok || connector == nil {
		return nil, errors.New("invalid data format")
	}
	return connector, nil
}

//add new connect
func (f *Group) AddConn(connId int64, conn *websocket.Conn) error {
	//check
	if connId <= 0 || conn == nil {
		return errors.New("invalid parameter")
	}

	//init new connector
	connector := NewConnector(connId, conn)

	//sync into bucket map with locker
	f.connMap.Store(connId, connector)

	//check and call the connected cb of outside
	if f.conf != nil && f.conf.CBForConnected != nil {
		f.conf.CBForConnected(f.conf.Uri, f.groupId, connId)
	}
	atomic.AddInt32(&f.connects, 1)

	return nil
}

////////////////
//private func
////////////////

//read loop
func (f *Group) readLoop() {
	var (
		m any = nil
	)
	//panic catch
	defer func() {
		if err := recover(); err != m {
			log.Printf("group %v read loop panic, err:%v\n", f.groupId, err)
		}
	}()

	//sub func for read connect message opt
	subRead := func(k, v interface{}) bool {
		var (
			data []byte
			err error
		)
		//detect connector
		conn, ok := v.(iface.IConnector)
		if ok && conn != nil {
			//read data
			data, err = conn.Read()
			if err != nil {
				if err == io.EOF {
					//lost or read a bad connect
					//remove and force close connect
					f.CloseConn(conn.GetConnId())
				}
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					//read timeout
					//do nothing
				}
			}else{
				//read data succeed
				//check and call the read cb of outside
				if f.conf != nil && f.conf.CBForRead != nil {
					f.conf.CBForRead(f.conf.Uri, f.groupId, conn.GetConnId(), data)
				}
			}
		}
		return true
	}

	//loop opt
	for {
		select {
		case <- f.readCloseChan:
			{
				//force quit read loop
				return
			}
		default:
			{
				//default map loop opt
				f.connMap.Range(subRead)

				//sleep awhile
				time.Sleep(10 * time.Millisecond)
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
	}()

	//sub func for write opt
	subWriteFunc := func(data *gvar.MsgData) error {
		//check
		if data == nil || data.Data == nil {
			return errors.New("invalid parameter")
		}

		if data.ConnIds != nil && len(data.ConnIds) > 0 {
			//send to assigned connect ids
			for _, connId := range data.ConnIds {
				if connId <= 0 {
					continue
				}
				conn, _ := f.GetConn(connId)
				if conn == nil {
					continue
				}
				//write to target conn
				conn.Write(data.Data)
			}
			return nil
		}

		//send to all connects
		subConnWriteFunc := func(k, v interface{}) bool {
			conn, ok := v.(iface.IConnector)
			if ok && conn != nil {
				conn.Write(data.Data)
			}
			return true
		}
		f.connMap.Range(subConnWriteFunc)
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
	f.connMap.Delete(connId)
}

//inter init
func (f *Group) interInit() {
	//init counter
	atomic.StoreInt32(&f.connects, 0)

	//spawn read and write loop
	go f.readLoop()
	go f.writeLoop()
}