package face

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"strconv"
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
 * dynamic websocket router face
 * - run batch dynamic groups
 */

//face info
type Dynamic struct {
	cfg          *gvar.GroupConf        //router origin conf reference
	connId       int64                  //inter atomic conn id counter
	groupMap     map[int64]iface.IGroup //dynamic group map
	sync.RWMutex
	Util
}

//construct
func NewDynamic(cfg *gvar.GroupConf) *Dynamic {
	this := &Dynamic{
		cfg: cfg,
		groupMap: map[int64]iface.IGroup{},
	}
	this.interInit()
	return this
}

//quit
func (f *Dynamic) Quit() {
	f.Lock()
	defer f.Unlock()
	for _, v := range f.groupMap {
		v.Quit()
	}
	//release old group map
	f.groupMap = nil
}

//get conf
func (f *Dynamic) GetConf() *gvar.GroupConf {
	return f.cfg
}

//remove group by id
func (f *Dynamic) RemoveGroup(groupId int64) error {
	//check
	if groupId <= 0 {
		return errors.New("invalid parameter")
	}

	//get old group
	oldGroup, _ := f.GetGroup(groupId)
	if oldGroup == nil {
		return errors.New("no such group")
	}
	oldGroup.Quit()

	//remove with locker
	f.Lock()
	defer f.Unlock()
	delete(f.groupMap, groupId)

	//hit gc rate
	gcRate := rand.Intn(define.FullPercent)
	needRebuildNewMap := false
	if gcRate > 0 && gcRate <= define.DynamicGroupGcRate {
		needRebuildNewMap = true
	}

	//gc opt
	if needRebuildNewMap || len(f.groupMap) <= 0 {
		//rebuild
		f.rebuild()
	}
	return nil
}

//get group by id
func (f *Dynamic) GetGroup(groupId int64) (iface.IGroup, error) {
	//check
	if groupId <= 0 {
		return nil, errors.New("invalid parameter")
	}
	if f.groupMap == nil {
		return nil, fmt.Errorf("group %v map is nil", groupId)
	}

	//get with locker
	f.RLock()
	defer f.RUnlock()
	v, ok := f.groupMap[groupId]
	if !ok || v == nil {
		return nil, errors.New("can't get group by id")
	}
	return v, nil
}

//create new group
//the new group should be pre-create
func (f *Dynamic) CreateGroup(groupId int64) (iface.IGroup, error) {
	//check
	if groupId <= 0 {
		return nil, errors.New("invalid parameter")
	}

	//get group first
	oldGroup, _ := f.GetGroup(groupId)
	if oldGroup != nil {
		return nil, errors.New("group had created")
	}

	//create new
	newGroup := NewGroup(groupId, f.cfg)

	//sync into env with locker
	f.Lock()
	defer f.Unlock()
	f.groupMap[groupId] = newGroup
	return newGroup, nil
}

//cast message
func (f *Dynamic) Cast(groupId int64, msg *gvar.MsgData) error {
	//check
	if groupId <= 0 || msg == nil || msg.Data == nil {
		return errors.New("invalid parameter")
	}

	//get target group
	targetGroup, err := f.GetGroup(groupId)
	if err != nil || targetGroup == nil {
		return err
	}

	//cast message to target group
	err = targetGroup.Cast(msg)
	return err
}

//websocket request entry
func (f *Dynamic) Entry(conn *websocket.Conn) {
	var (
		newConnId int64
	)
	//check
	if conn == nil {
		return
	}

	//check group id para
	groupId, err := f.getAndVerifyGroupId(conn)
	if err != nil || groupId <= 0 {
		log.Printf("group %v, verify group id frailed", groupId)
		return
	}

	//gen new connect id
	if f.cfg.CBForGenConnId != nil {
		newConnId = f.cfg.CBForGenConnId()
	}else{
		newConnId = atomic.AddInt64(&f.connId, 1)
	}
	if newConnId <= 0 {
		log.Printf("group %v, can't gen new connect id\n", groupId)
		return
	}

	//get or create group
	groupObj, subErr := f.GetGroup(groupId)
	if subErr != nil {
		log.Printf("group %v, get group failed, err:%v\n", groupId, subErr.Error())
		return
	}
	if groupObj == nil {
		log.Printf("group %v, can't get group object\n", groupId)
		return
	}

	//add new connect into target bucket
	groupObj.AddConn(newConnId, conn)

	//keep the new connect active
	select {}
}

////////////////
//private func
////////////////

//rebuild
func (f *Dynamic) rebuild() {
	//init new group map and release old map
	newGroupMap := map[int64]iface.IGroup{}
	for k, v := range f.groupMap {
		newGroupMap[k] = v
	}
	f.groupMap = newGroupMap

	//force gc
	runtime.GC()
}

//get and verify group id para
func (f *Dynamic) getAndVerifyGroupId(conn *websocket.Conn) (int64, error) {
	//get group id from path para
	groupId, err := f.GetPathPara(conn, define.PathParaNameOfGroup)
	if err != nil {
		return 0, err
	}

	//convert
	groupIdInt, _ := strconv.ParseInt(groupId, 10, 64)
	if groupIdInt <= 0 {
		return 0, errors.New("invalid group id")
	}

	//check the cb for verify group and run it
	if f.cfg != nil && f.cfg.CBForVerifyGroup != nil {
		err = f.cfg.CBForVerifyGroup(f.cfg.Uri, groupIdInt)
		if err != nil {
			return 0, err
		}
		return groupIdInt, nil
	}
	return groupIdInt, nil
}

//inter init
func (f *Dynamic) interInit() {
	//init inter counter
	atomic.StoreInt64(&f.connId, 0)
}