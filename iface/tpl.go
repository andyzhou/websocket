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
	SetMainTpl(tag, file string) error
	SetGlobalTplFiles(files ... string) bool
	SetAutoLoad(auto bool)
}
