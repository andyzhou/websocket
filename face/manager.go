package face

import "github.com/andyzhou/websocket/iface"

/*
 * face of inter manager, implement of IManager
 *
 * - manager tpl
 */

//inter macro define
const (
	tplRootPath = "./tpl"
)

//face info
type Manager struct {
	tplRootPath string
	tplMap map[string]iface.ITpl //mainTpl -> ITpl
}

//construct
func NewManager() *Manager {
	//self init
	this := &Manager{
		tplRootPath: tplRootPath,
		tplMap:make(map[string]iface.ITpl),
	}
	return this
}

//set tpl root path
func (f *Manager) SetTplRootPath(rootPath string) bool {
	if rootPath == "" {
		return false
	}
	f.tplRootPath = rootPath
	return true
}

