package face

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"

	"github.com/andyzhou/websocket/define"
	"github.com/andyzhou/websocket/gvar"
	"github.com/andyzhou/websocket/iface"

	"golang.org/x/net/websocket"
)

/*
 * websocket router face
 */

//face info
type Router struct {
	cfg       *gvar.RouterConf      //router origin conf reference
	connId    int64                 //inter atomic conn id counter
	buckets   int                   //total buckets
	bucketMap map[int]iface.IBucket //bucket map
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
}

//get router config
func (f *Router) GetConf() *gvar.RouterConf {
	return f.cfg
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

	//pick rand bucket by conn id
	randBucketIdx := int(newConnId % int64(f.buckets))

	//get target bucket with locker
	targetBucket, err := f.getBucket(randBucketIdx)
	if err != nil || targetBucket == nil {
		log.Printf("router %v, can't get target bucket\n", f.cfg.Uri)
		return
	}

	//add new connect into target bucket
	targetBucket.AddConn(newConnId, conn)

	//keep the new connect active
	select {}
}

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

//inter init
func (f *Router) interInit() {
	//setup total buckets
	f.buckets = f.cfg.Buckets
	if f.buckets <= 0 {
		f.buckets = define.DefaultBuckets
	}

	//init inter buckets
	f.Lock()
	defer f.Unlock()
	for i := 0; i < f.buckets; i++ {
		bucket := NewBucket(i, f.cfg)
		f.bucketMap[i] = bucket
	}
}