package face

import (
	"fmt"
	"html/template"
)

/*
 * face of template, implement of ITpl
 */

//face info
type Tpl struct {
	rootPath string
	tplFiles []string
	tpl *template.Template
}

//construct
func NewTpl(rootPath string) *Tpl {
	//self init
	this := &Tpl{
		rootPath: rootPath,
		tpl: new(template.Template),
		tplFiles: make([]string, 0),
	}
	return this
}

//reset all tpl files
func (f *Tpl) ResetTpl() {
	f.tplFiles = make([]string, 0)
	f.tpl = new(template.Template)
}

//parse and execute tpl
func (f *Tpl) Execute() {
	f.tpl = template.Must(f.tpl.ParseFiles(f.tplFiles...))
}

//add tpl file
func (f *Tpl) AddTpl(file string) bool {
	if file == "" {
		return false
	}
	tplFilePath := fmt.Sprintf("%s/%s", f.rootPath, file)
	f.tplFiles = append(f.tplFiles, tplFilePath)
	return true
}


