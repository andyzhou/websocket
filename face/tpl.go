package face

import (
	"fmt"
	"html/template"
	"net/http"
)

/*
 * face of template, implement of ITpl
 */

//face info
type Tpl struct {
	autoLoad bool
	rootPath string
	tplFiles []string
	tpl *template.Template
}

//construct
func NewTpl(rootPath string) *Tpl {
	//self init
	this := &Tpl{
		rootPath: rootPath,
		autoLoad: true,
		tpl: new(template.Template),
		tplFiles: make([]string, 0),
	}
	return this
}

//reset all tpl files
func (f *Tpl) ResetTpl() {
	f.tplFiles = make([]string, 0)
}

//set auto load switch
func (f *Tpl) SetAutoLoad(auto bool) {
	f.autoLoad = auto
}

//parse and execute tpl
func (f *Tpl) Execute(
			mainTpl string,
			data interface{},
			w http.ResponseWriter,
			r *http.Request,
		) {
	var (
		err error
	)

	//parse tpl files
	f.tpl = template.New(mainTpl)
	f.tpl, err = template.ParseFiles(f.tplFiles...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//execute
	err = f.tpl.ExecuteTemplate(w, mainTpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//add tpl file
func (f *Tpl) AddTpl(files ...string) bool {
	var (
		tplFilePath string
	)

	//basic check
	if files == nil || len(files) <= 0 {
		return false
	}

	//add batch tpl file
	for _, file := range files {
		tplFilePath = fmt.Sprintf("%s/%s", f.rootPath, file)
		f.tplFiles = append(f.tplFiles, tplFilePath)
	}

	return true
}


