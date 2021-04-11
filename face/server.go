package face

import (
	"fmt"
	"github.com/andyzhou/websocket/define"
	"github.com/andyzhou/websocket/iface"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

/*
 * face of web socket server, implement of IWServer
 */

//face info
type Server struct {
	port int
	address string //host:port
	staticPath string //static root path
	tplPath string //tpl root path
	globalTplFiles []string
	hsm *http.ServeMux
	router *mux.Router
	routerMap *sync.Map //channel -> IRouter
	routerCount int64
	connId int64
}

//construct
func NewWServer(port int) *Server {
	//self init
	this := &Server{
		port: port,
		address: fmt.Sprintf(":%d", port),
		hsm: http.NewServeMux(),
		router: mux.NewRouter(),
		routerMap: new(sync.Map),
		globalTplFiles: make([]string, 0),
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

//set static root path
func (f *Server) SetStaticPath(staticPath string) {
	f.staticPath = staticPath
}

//set tpl root path
func (f *Server) SetTplPath(tplPath string) {
	f.tplPath = tplPath
}

//set global tpl file
func (f *Server) SetGlobalTplFile(tplFile ... string) bool {
	//basic check
	if tplFile == nil || len(tplFile) <= 0 {
		return false
	}

	//reset global tpl files
	f.globalTplFiles = make([]string, 0)
	f.globalTplFiles = append(f.globalTplFiles, tplFile...)
	return true
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

//register http page router
func (f *Server) RegisterPageRouter(subUrl string) bool  {
	//basic check
	if subUrl == "" {
		return false
	}

	//page router
	f.router.HandleFunc(subUrl, f.interTplPageRouter).Methods(define.RouterMethodOfGet)

	return true
}

//register http static router
func (f *Server) RegisterStaticRouter(subUrl string) bool {
	//basic check
	if subUrl == "" {
		return false
	}

	//static file
	f.router.HandleFunc(subUrl, f.interStaticFileRouter).Methods(define.RouterMethodOfGet)

	return true
}

/////////////////
//private func
/////////////////

//inter tpl page router
func (f *Server) interTplPageRouter(
					w http.ResponseWriter,
					r *http.Request,
				) {
	//get page file name
	params := mux.Vars(r)
	pageName, ok := params[define.PageParaName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	//init main tpl file
	mainTplFile := fmt.Sprintf("%s%s", pageName, define.TplFileExtName)

	//check tpl root path
	if f.tplPath == "" {
		//get root path
		rootPath, _ := os.Getwd()
		f.tplPath = rootPath
	}

	//init tpl full root path
	rootPath, _ := os.Getwd()
	tplRootPath := fmt.Sprintf("%s/%s", rootPath, f.tplPath)

	//check main tpl is exist or not
	mainTplPath := fmt.Sprintf("%s/%s", tplRootPath, mainTplFile)
	isExists := f.checkFileIsExists(mainTplPath)
	if !isExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	//init tpl instance
	tpl := NewTpl(tplRootPath)

	//add global tpl
	if f.globalTplFiles != nil {
		tpl.AddTpl(f.globalTplFiles...)
	}

	//add main tpl
	tpl.AddTpl(mainTplFile)

	//execute tpl
	tpl.Execute(mainTplFile, nil, w, r)
}

//inter static file router
func (f *Server) interStaticFileRouter(
					w http.ResponseWriter,
					r *http.Request,
				) {
	//get static file name
	params := mux.Vars(r)
	fileName, ok := params[define.FileParaName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	//get root path
	rootPath, _ := os.Getwd()
	staticFile := fmt.Sprintf("%s/%s/%s", rootPath, f.staticPath, fileName)

	//read file
	byteData, err := ioutil.ReadFile(staticFile)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fileType := http.DetectContentType(byteData)

	//set access control
	w.Header().Set("Access-Control-Allow-Origin", "*")

	isJs := strings.HasSuffix(staticFile, ".js")
	if isJs {
		//set http header
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	}else{
		//set http header
		w.Header().Set("Content-Type", fileType)
	}

	w.Write(byteData)
}

//check file exist or not
func (f *Server) checkFileIsExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

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