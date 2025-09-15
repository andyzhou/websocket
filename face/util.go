package face

import (
	"bytes"
	"encoding/gob"
	"errors"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
	"math/rand"
	"net/url"
	"reflect"
	"time"
	"unsafe"
)

/*
 * @author <AndyZhou>
 * @mail <diudiu8848@163.com>
 * util tools
 */

//face info
type Util struct {}

//get query para
func (f *Util) GetQueryParas(conn *websocket.Conn) (url.Values, error) {
	//check
	if conn == nil {
		return nil, errors.New("invalid parameter")
	}

	//get origin request paras
	httpReq := conn.Request()
	queryParas := httpReq.URL.Query()
	return queryParas, nil
}

//get path para
func (f *Util) GetPathPara(conn *websocket.Conn, paraName string) (string, error) {
	//check
	if conn == nil || paraName == "" {
		return "", errors.New("invalid parameter")
	}

	//get all path params
	pathParams := mux.Vars(conn.Request())
	if pathParams == nil || len(pathParams) <= 0 {
		return "", nil
	}

	//get para value
	paraVal, _ := pathParams[paraName]
	return paraVal, nil
}

//check chan is closed or not
//true:closed, false:opening
func (f *Util) IsChanClosed(ch interface{}) (bool, error) {
	//check
	if reflect.TypeOf(ch).Kind() != reflect.Chan {
		return false, errors.New("input value not channel type")
	}

	// get interface value pointer, from cgo_export
	// typedef struct { void *t; void *v; } GoInterface;
	// then get channel real pointer
	cPtr := *(*uintptr)(unsafe.Pointer(
		unsafe.Pointer(uintptr(unsafe.Pointer(&ch)) + unsafe.Sizeof(uint(0))),
	))

	// this function will return true if chan.closed > 0
	// see hchan on https://github.com/golang/go/blob/master/src/runtime/chan.go
	// type hchan struct {
	// qcount   uint           // total data in the queue
	// dataqsiz uint           // size of the circular queue
	// buf      unsafe.Pointer // points to an array of dataqsiz elements
	// elemsize uint16
	// closed   uint32
	// **

	cPtr += unsafe.Sizeof(uint(0))*2
	cPtr += unsafe.Sizeof(unsafe.Pointer(uintptr(0)))
	cPtr += unsafe.Sizeof(uint16(0))
	return *(*uint32)(unsafe.Pointer(cPtr)) > 0, nil
}

//deep copy object
func (f *Util) DeepCopy(src, dist interface{}) (err error){
	buf := bytes.Buffer{}
	if err = gob.NewEncoder(&buf).Encode(src); err != nil {
		return
	}
	return gob.NewDecoder(&buf).Decode(dist)
}

//shuffle objs
func (f *Util) Shuffle(slice interface{}) {
	rand.Seed(time.Now().UnixNano())
	rv := reflect.ValueOf(slice)
	swap := reflect.Swapper(slice)
	length := rv.Len()
	for i := length - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		swap(i, j)
	}
}