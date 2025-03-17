package websocket

import (
	"errors"
	"fmt"
	"github.com/andyzhou/websocket/define"
	"net/http"
	"runtime"
	"sync"

	"github.com/andyzhou/websocket/face"
	"github.com/andyzhou/websocket/gvar"
	"github.com/andyzhou/websocket/iface"

	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * websocket server face
 */

//face info
type Server struct {
	port          int
	hsm           *http.ServeMux //mux http server
	router        *mux.Router
	routerMap     map[string]iface.IRouter  //persistent routers, uri -> IRouter
	dynamicMap    map[string]iface.IDynamic //dynamic groups, uri -> IDynamic
	wg            sync.WaitGroup
	dynamicLocker sync.RWMutex
	sync.RWMutex
}

//construct
func NewServer() *Server {
	this := &Server{
		hsm: http.NewServeMux(),
		router: mux.NewRouter(),
		routerMap: map[string]iface.IRouter{},
		dynamicMap: map[string]iface.IDynamic{},
	}
	return this
}

//quit
func (f *Server) Quit() {
	//force close router
	f.Lock()
	defer f.Unlock()
	for k, v := range f.routerMap {
		v.Quit()
		delete(f.routerMap, k)
	}

	//force close dynamic groups
	f.dynamicLocker.Lock()
	defer f.dynamicLocker.Unlock()
	for k, v := range f.dynamicMap {
		v.Quit()
		delete(f.dynamicMap, k)
	}

	//gc opt
	runtime.GC()
	f.wg.Done()
}

//start
func (f *Server) Start(port int) error {
	//check
	if port <= 0 {
		return errors.New("invalid parameter")
	}

	//format and handle address
	address := fmt.Sprintf(":%v", port)
	f.hsm.Handle("/", f.router)

	//listen port
	f.wg.Add(1)
	go http.ListenAndServe(address, f.hsm)

	f.wg.Wait()
	return nil
}

//get all routers
func (f *Server) GetAllRouters() map[string]iface.IRouter {
	return f.routerMap
}

//get all dynamics
func (f *Server) GetAllDynamics() map[string]iface.IDynamic {
	return f.dynamicMap
}

//get router by uri
func (f *Server) GetRouter(uri string) (iface.IRouter, error) {
	//check
	if uri == "" {
		return nil, errors.New("invalid parameter")
	}

	//get by uri with locker
	f.Lock()
	defer f.Unlock()
	v, ok := f.routerMap[uri]
	if !ok || v == nil {
		return nil, errors.New("no such router")
	}
	return v, nil
}

//get dynamic by uri
func (f *Server) GetDynamic(uri string) (iface.IDynamic, error) {
	//check
	if uri == "" {
		return nil, errors.New("invalid parameter")
	}

	//get by uri with locker
	f.dynamicLocker.Lock()
	defer f.dynamicLocker.Unlock()
	v, ok := f.dynamicMap[uri]
	if !ok || v == nil {
		return nil, errors.New("no such dynamic")
	}
	return v, nil
}

//register new dynamic router
func (f *Server) RegisterDynamic(cfg *gvar.GroupConf) (iface.IDynamic, error) {
	//check
	if cfg == nil || cfg.Uri == "" {
		return nil, errors.New("invalid parameter")
	}

	//check old
	oldDynamic, _ := f.GetDynamic(cfg.Uri)
	if oldDynamic != nil {
		return oldDynamic, errors.New("this uri dynamic had exists")
	}

	//init new sub dynamic face
	subDynamic := face.NewDynamic(cfg)

	//format dynamic uri with path para info
	//path para value used as group id
	uriWithPathPara := fmt.Sprintf("%v/{%v}", cfg.Uri, define.GroupPathParaName)

	//add websocket sub router handle
	f.router.Handle(uriWithPathPara, websocket.Handler(subDynamic.Entry))

	//sync into running map with locker
	f.Lock()
	defer f.Unlock()
	f.dynamicMap[cfg.Uri] = subDynamic
	return subDynamic, nil
}

//register new persistent router
func (f *Server) RegisterRouter(cfg *gvar.RouterConf) error {
	//check
	if cfg == nil || cfg.Uri == "" {
		return errors.New("invalid parameter")
	}

	//check old
	oldRouter, _ := f.GetRouter(cfg.Uri)
	if oldRouter != nil {
		return errors.New("this uri router had exists")
	}

	//init new sub router face
	subRouter := face.NewRouter(cfg)

	//add websocket sub router handle
	f.router.Handle(cfg.Uri, websocket.Handler(subRouter.Entry))

	//sync into running map with locker
	f.Lock()
	defer f.Unlock()
	f.routerMap[cfg.Uri] = subRouter
	return nil
}

//generate message data
func (f *Server) GenMsgData() *gvar.MsgData {
	return &gvar.MsgData{}
}

//generate group config
func (f *Server) GenGroupCfg() *gvar.GroupConf {
	return &gvar.GroupConf{}
}

//generate router config
func (f *Server) GenRouterCfg() *gvar.RouterConf {
	return &gvar.RouterConf{}
}