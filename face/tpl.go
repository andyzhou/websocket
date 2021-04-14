package face

import (
	"fmt"
	"html/template"
	"net/http"
)

/*
 * face of template, implement of ITpl
 *
 * - support cached tpl for performance
 * - sub template instance manage self
 */

//face info
type Tpl struct {
	autoLoad bool
	rootPath string
	tplFiles []string
	tpl *template.Template
	globalTplFiles []string
	tplMap map[string]*template.Template //mainTpl -> *template
}

//construct
func NewTpl(rootPath string) *Tpl {
	//self init
	this := &Tpl{
		rootPath: rootPath,
		autoLoad: true,
		tpl: new(template.Template),
		tplFiles: make([]string, 0),
		globalTplFiles:make([]string, 0),
		tplMap:make(map[string]*template.Template),
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

//set main tpl file
func (f *Tpl) SetMainTpl(tag, file string) bool {
	//basic check
	if tag == "" || file == "" {
		return false
	}


	return true
}

//set global tpl files
func (f *Tpl) SetGlobalTplFiles(files ... string) bool {
	//basic check
	if files == nil || len(files) <= 0 {
		return false
	}

	//reset global tpl files
	f.globalTplFiles = make([]string, 0)
	f.globalTplFiles = files
	return true
}


//////////////////
//private func
//////////////////

func (f *Tpl) getTemplateByTag(tag string) *template.Template {
	//basic check
	if tag == "" || f.tplMap == nil {
		return nil
	}
	v, ok := f.tplMap[tag]
	if !ok {
		return nil
	}
	return v
}