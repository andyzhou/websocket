package face

import (
	"errors"
	"fmt"
	"io"
	"log"
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
	conf           *gvar.RouterConf //reference of parent router
	connMap        sync.Map         //connId -> IConnector
	connects       int64            //total connects
	writeChan      chan gvar.MsgData
	readCloseChan  chan bool
	writeCloseChan chan bool
	Util
}

//construct
func NewBucket(bucketId int, cfg *gvar.RouterConf) *Bucket {
	this := &Bucket{
		bucketId: bucketId,
		conf: cfg,
		connMap: sync.Map{},
		writeChan: make(chan gvar.MsgData, define.DefaultBucketWriteChan),
		readCloseChan: make(chan bool, 1),
		writeCloseChan: make(chan bool, 1),
	}
	this.interInit()
	return this
}

//quit
func (f *Bucket) Quit() {
	//force close main loop
	close(f.writeCloseChan)
	close(f.readCloseChan)

	//force close with locker
	rangeOpt := func(k, v interface{}) bool {
		conn, ok := v.(iface.IConnector)
		if ok && conn != nil {
			conn.Close()
		}
		f.connMap.Delete(k)
		return true
	}
	f.connMap.Range(rangeOpt)

	//gc opt
	runtime.GC()
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
	connector, err := f.GetConn(connId)
	if err != nil || connector == nil {
		return err
	}

	//force close connect
	connector.Close()

	//remove conn from map
	f.connMap.Delete(connId)
	atomic.AddInt64(&f.connects, -1)
	if f.connects <= 0 {
		atomic.StoreInt64(&f.connects, 0)
		//gc opt
		runtime.GC()
	}

	//check and call the closed cb of outside
	if f.conf != nil && f.conf.CBForClosed != nil {
		f.conf.CBForClosed(f.conf.Uri, connId)
	}
	return nil
}

//get old connect
func (f *Bucket) GetConn(connId int64) (iface.IConnector, error) {
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
func (f *Bucket) AddConn(connId int64, conn *websocket.Conn) error {
	//check
	if connId <= 0 || conn == nil {
		return errors.New("invalid parameter")
	}

	//init new connector
	connector := NewConnector(connId, conn)

	//sync into bucket map with locker
	f.connMap.Store(connId, connector)
	atomic.AddInt64(&f.connects, 1)

	//check and call the connected cb of outside
	if f.conf != nil && f.conf.CBForConnected != nil {
		f.conf.CBForConnected(f.conf.Uri, connId)
	}
	return nil
}

////////////////
//private func
////////////////

//read loop
func (f *Bucket) readLoop() {
	var (
		m any = nil
	)
	//panic catch
	defer func() {
		if err := recover(); err != m {
			log.Printf("bucket %v read loop panic, err:%v\n", f.bucketId, err)
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
					f.conf.CBForRead(f.conf.Uri, conn.GetConnId(), data)
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
				//if conn.GetEntrustGroup() > 0 {
				//	continue
				//}

				//write to target conn
				conn.Write(data.Data)
			}
			return nil
		}

		//send to all connects
		subConnWriteFunc := func(k, v interface{}) bool {
			conn, ok := v.(iface.IConnector)
			if ok && conn != nil {
				//if conn.GetEntrustGroup() > 0 {
				//	return true
				//}
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
func (f *Bucket) removeConnect(connId int64) {
	if connId <= 0 {
		return
	}
	f.connMap.Delete(connId)
}

//inter init
func (f *Bucket) interInit() {
	//spawn read and write loop
	go f.readLoop()
	go f.writeLoop()
}
