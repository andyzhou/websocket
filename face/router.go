package face

import (
	"errors"
	"log"
	"strconv"
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
 * - multi buckets contain connections
 */

//face info
type Router struct {
	cfg         *gvar.RouterConf       //router origin conf reference
	connId      int64                  //inter atomic conn id counter
	buckets     int                    //total buckets of config
	bucketMap   map[int]iface.IBucket  //bucket map container
	Util
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
	for k, v := range f.bucketMap {
		v.Quit()
		f.bucketMap[k] = nil
	}
	f.bucketMap = nil
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

//set connect owner id
func (f *Router) SetOwner(connId, ownerId int64) error {
	//check
	if connId <= 0 || ownerId <= 0 {
		return errors.New("invalid parameter")
	}

	//get target bucket
	bucket, err := f.getBucketByConnId(connId)
	if err != nil || bucket == nil {
		if err != nil {
			return err
		}
		return errors.New("can't get bucket by id")
	}

	//set connector owner
	err = bucket.SetOwner(connId, ownerId)
	return err
}

//broad cast data
func (f *Router) Cast(msg *gvar.MsgData) error {
	//check
	if msg == nil || msg.Data == nil {
		return errors.New("invalid parameter")
	}

	//cast to assigned buckets
	if len(msg.BucketIds) > 0 {
		for _, idx := range msg.BucketIds {
			v, ok := f.bucketMap[idx]
			if ok && v != nil {
				v.Broadcast(msg)
			}
		}
		return nil
	}

	//cast to all buckets
	for _, v := range f.bucketMap {
		v.Broadcast(msg)
	}
	return nil
}

//websocket request entry
//ws conn init first time
func (f *Router) Entry(conn *websocket.Conn) {
	var (
		newConnId    int64
		bucketId     int
		targetBucket iface.IBucket
		err          error
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

	if f.cfg.BucketIdPara != "" {
		//get assigned bucket id pass request para
		queryParas, _ := f.GetQueryParas(conn)
		bucketIdVal := queryParas.Get(f.cfg.BucketIdPara)
		bucketId, _ = strconv.Atoi(bucketIdVal)
	}

	if bucketId > 0 {
		//get target by id
		targetBucket, err = f.getBucket(bucketId)
	}else{
		//get target bucket by conn id
		targetBucket, err = f.getBucketByConnId(newConnId)
	}
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

	//get bucket by id
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

	//pick target bucket idx by conn id
	targetBucketIdx := int(connId % int64(f.buckets))

	//get target bucket with locker
	targetBucket, err := f.getBucket(targetBucketIdx)
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

	//init inter buckets container
	for i := 0; i < f.buckets; i++ {
		bucket := NewBucket(f, i, f.cfg)
		f.bucketMap[i] = bucket
	}
}