package face

import (
	"errors"
	"log"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/andyzhou/websocket/define"
	"github.com/andyzhou/websocket/gvar"
	"github.com/andyzhou/websocket/iface"
	"golang.org/x/net/websocket"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * persistent websocket router face
 */

//face info
type Router struct {
	cfg         *gvar.RouterConf       //router origin conf reference
	connId      int64                  //inter atomic conn id counter
	buckets     int                    //total buckets
	bucketMap   map[int]iface.IBucket  //bucket map
	sync.RWMutex
}

//construct
func NewRouter(cfg *gvar.RouterConf) *Router {
	this := &Router{
		cfg: cfg,
		bucketMap: map[int]iface.IBucket{},
	}
	this.interInit()
	return this
}

//quit
func (f *Router) Quit()  {
	//clear buckets
	f.Lock()
	defer f.Unlock()
	for k, v := range f.bucketMap {
		v.Quit()
		delete(f.bucketMap, k)
	}

	//gc opt
	runtime.GC()
}

//get router config
func (f *Router) GetConf() *gvar.RouterConf {
	return f.cfg
}

//get connector by id
func (f *Router) GetConnector(connId int64) (iface.IConnector, error) {
	//check
	if connId <= 0 {
		return nil, errors.New("invalid parameter")
	}

	//get target bucket by conn id
	targetBucket, err := f.getBucketByConnId(connId)
	if err != nil {
		return nil, err
	}
	if targetBucket == nil {
		return nil, errors.New("can't get target bucket")
	}

	//get target connector
	return targetBucket.GetConn(connId)
}

//broad cast data
func (f *Router) Cast(msg *gvar.MsgData) error {
	//check
	if msg == nil || msg.Data == nil {
		return errors.New("invalid parameter")
	}

	//cast to all buckets with locker
	f.Lock()
	defer f.Unlock()
	for _, v := range f.bucketMap {
		v.Broadcast(msg)
	}
	return nil
}

//websocket request entry
func (f *Router) Entry(conn *websocket.Conn) {
	var (
		newConnId int64
	)
	//check
	if conn == nil {
		return
	}

	//gen new connect id
	if f.cfg.CBForGenConnId != nil {
		newConnId = f.cfg.CBForGenConnId()
	}else{
		newConnId = atomic.AddInt64(&f.connId, 1)
	}
	if newConnId <= 0 {
		log.Printf("router %v, can't gen new connect id\n", f.cfg.Uri)
		return
	}

	//get target bucket by conn id
	targetBucket, err := f.getBucketByConnId(newConnId)
	if err != nil || targetBucket == nil {
		log.Printf("router %v, can't get target bucket\n", f.cfg.Uri)
		return
	}

	//add new connect into target bucket
	targetBucket.AddConn(newConnId, conn)

	//keep the new connect active
	select {}
}

////////////////
//private func
////////////////

//get bucket by idx
func (f *Router) getBucket(idx int) (iface.IBucket, error) {
	//check
	if idx < 0 {
		return nil, errors.New("invalid parameter")
	}

	//get with locker
	f.Lock()
	defer f.Unlock()
	v, ok := f.bucketMap[idx]
	if !ok || v == nil {
		return nil, errors.New("no bucket by idx")
	}
	return v, nil
}

//get target bucket by conn id
func (f *Router) getBucketByConnId(connId int64) (iface.IBucket, error) {
	//check
	if connId <= 0 {
		return nil, errors.New("invalid parameter")
	}

	//pick rand bucket by conn id
	randBucketIdx := int(connId % int64(f.buckets))

	//get target bucket with locker
	targetBucket, err := f.getBucket(randBucketIdx)
	return targetBucket, err
}

//inter init
func (f *Router) interInit() {
	//init inter counter
	atomic.StoreInt64(&f.connId, 0)

	//setup total buckets
	f.buckets = f.cfg.Buckets
	if f.buckets <= 0 {
		f.buckets = define.DefaultBuckets
	}

	//init inter buckets
	f.Lock()
	defer f.Unlock()
	for i := 0; i < f.buckets; i++ {
		bucket := NewBucket(f, i, f.cfg)
		f.bucketMap[i] = bucket
	}
}