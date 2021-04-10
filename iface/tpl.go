package iface

/*
 * interface of template
 */

type ITpl interface {
	ResetTpl()
	Execute()
	AddTpl(file string) bool
}
