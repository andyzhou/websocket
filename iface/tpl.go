package iface

import "net/http"

/*
 * interface of template
 */

type ITpl interface {
	ResetTpl()
	Execute(
			mainTpl string,
			data interface{},
			w http.ResponseWriter,
			r *http.Request,
		)
	AddTpl(file ...string) bool //this will be removed
	SetMainTpl(tag, file string) bool
	SetGlobalTplFiles(files ... string) bool
	SetAutoLoad(auto bool)
}
