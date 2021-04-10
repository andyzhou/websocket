package face

import (
	"fmt"
	"github.com/andyzhou/websocket/define"
	"github.com/andyzhou/websocket/iface"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
	"net/http"
	"sync"
	"sync/atomic"
)

/*
 * face of server, implement of IServer
 */

//face info
type Server struct {
	port int
	address string //host:port
	hsm *http.ServeMux
	router *mux.Router
	routerMap *sync.Map //channel -> IRouter
	routerCount int64
	connId int64
}

//construct
func NewServer(port int) *Server {
	//self init
	this := &Server{
		port: port,
		address: fmt.Sprintf(":%d", port),
		hsm: http.NewServeMux(),
		router: mux.NewRouter(),
		routerMap: new(sync.Map),
	}
	return this
}

//quit
func (f *Server) Quit() {
	//clean up channel
	cf := func(k, v interface{}) bool {
		router, ok := v.(iface.IRouter)
		if !ok {
			return false
		}
		router.Quit()
		return true
	}
	f.routerMap.Range(cf)
}

//start server
func (f *Server) Start() {
	f.hsm.Handle("/", f.router)
	go http.ListenAndServe(f.address, f.hsm)
}

//get web socket parameters
func (f *Server) GetWsParas(ws *websocket.Conn) map[string]string {
	if ws == nil {
		return nil
	}
	return mux.Vars(ws.Request())
}

//get new connect id
func (f *Server) GetNewConnId() int64 {
	return atomic.AddInt64(&f.connId, 1)
}

//get total router
func (f *Server) GetTotalRouter() int64 {
	return f.routerCount
}

//get IRouter by tag
func (f *Server) GetRouter(channel string) iface.IRouter {
	//basic check
	if channel == "" || f.routerMap == nil {
		return nil
	}
	return f.getRouter(channel)
}

//register web sock¡et router
func (f *Server) RegisterWSRouter(
						subUrl string,
						channelName string,
						userRouter iface.IUserRouter,
					) bool {
	//basic check
	if subUrl == "" || channelName == "" || userRouter == nil {
		return false
	}

	//init new router
	router := f.createRouter(channelName, userRouter)

	//add web socket sub router
	f.router.Handle(subUrl, websocket.Handler(router.Entry))

	return true
}

//register http router
func (f *Server) RegisterHttpRouter(
						subUrl string,
						cb func(w http.ResponseWriter, r *http.Request),
						method ... string,
					) bool {
	//basic check
	if subUrl == "" || cb == nil {
		return false
	}

	if method == nil {
		method = []string{
			define.RouterMethodOfGet,
		}
	}

	//add http sub router
	f.router.HandleFunc(subUrl, cb).Methods(method...)

	return true
}

/////////////////
//private func
/////////////////

//check or create router
func (f *Server) createRouter(
						tag string,
						userRouter iface.IUserRouter,
					) iface.IRouter {
	//basic check
	if tag == "" || userRouter == nil {
		return nil
	}

	//check old
	old := f.getRouter(tag)
	if old != nil {
		return old
	}

	//create new router
	router := NewRouter(tag, userRouter)

	//sync into map
	f.routerMap.Store(tag, router)
	atomic.AddInt64(&f.routerCount, 1)

	return router
}

//get IRouter by tag
func (f *Server) getRouter(tag string) iface.IRouter {
	//basic check
	if tag == "" || f.routerMap == nil {
		return nil
	}
	v, ok := f.routerMap.Load(tag)
	if !ok {
		return nil
	}
	router, ok := v.(iface.IRouter)
	if !ok {
		return nil
	}
	return router
}