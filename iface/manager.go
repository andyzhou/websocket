package iface

/*
 * interface of inter manager
 */

type IManager interface {
	//set tpl root path
	SetTplRootPath(rootPath string) bool
}
