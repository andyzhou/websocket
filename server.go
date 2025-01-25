package websocket

import (
	"errors"
	"fmt"
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
 * websocket server face
 */

//face info
type Server struct {
	port      int
	hsm       *http.ServeMux //mux http server
	router    *mux.Router
	routerMap map[string]iface.IRouter //uri -> IRouter
	sync.RWMutex
}

//construct
func NewServer() *Server {
	this := &Server{
		hsm: http.NewServeMux(),
		router: mux.NewRouter(),
		routerMap: map[string]iface.IRouter{},
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

	//gc opt
	runtime.GC()
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
	go http.ListenAndServe(address, f.hsm)

	return nil
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

//register new router
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

//generate router config
func (f *Server) GenRouterCfg() *gvar.RouterConf {
	return &gvar.RouterConf{}
}