package face

import (
	"fmt"
	"github.com/pkg/errors"
	"html/template"
	"net/http"
)

/*
 * face of template, implement of ITpl
 *
 * - support cached tpl for performance
 * - sub template instance manage self
 */

//main tpl info
type mainTplInfo struct {
	mainTplFile string
	tpl *template.Template
}

//face info
type Tpl struct {
	autoLoad bool
	rootPath string
	tplFiles []string
	tpl *template.Template
	globalTplFiles []string
	tplMap map[string]*mainTplInfo //mainTpl -> *mainTplInfo
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
		tplMap:make(map[string]*mainTplInfo),
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
			tag string,
			data interface{},
			w http.ResponseWriter,
			r *http.Request,
		) {
	var (
		err error
	)

	//check tpl info
	tplInfo := f.getTemplateByTag(tag)
	if tplInfo == nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if f.autoLoad {
		mainTplPath := fmt.Sprintf("%s/%s", f.rootPath, tplInfo.mainTplFile)
		files := f.globalTplFiles
		files = append(files, mainTplPath)
		tplInfo.tpl, err = template.ParseFiles(files...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	//execute
	err = tplInfo.tpl.ExecuteTemplate(w, tag, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//set main tpl file
func (f *Tpl) SetMainTpl(tag, file string) error {
	var (
		isNew bool
		err error
	)

	//basic check
	if tag == "" || file == "" {
		return errors.New("invalid parameter")
	}

	//check tpl info
	tplInfo := f.getTemplateByTag(tag)
	if tplInfo == nil {
		//init new
		tplInfo = &mainTplInfo{
			mainTplFile:file,
			tpl:template.New(tag),
		}
		//sync into running env
		f.tplMap[tag] = tplInfo
		isNew = true
	}

	if isNew || f.autoLoad {
		//need pre-parse tpl files
		mainTplPath := fmt.Sprintf("%s/%s", f.rootPath, file)
		files := f.globalTplFiles
		files = append(files, mainTplPath)
		tplInfo.tpl, err = template.ParseFiles(files...)
		if err != nil {
			return err
		}
	}

	return nil
}

//set global tpl files
func (f *Tpl) SetGlobalTplFiles(files ... string) bool {
	var (
		tplFilePath string
	)

	//basic check
	if files == nil || len(files) <= 0 {
		return false
	}

	//reset global tpl files
	f.globalTplFiles = make([]string, 0)

	//add batch tpl file
	for _, file := range files {
		tplFilePath = fmt.Sprintf("%s/%s", f.rootPath, file)
		f.globalTplFiles = append(f.globalTplFiles, tplFilePath)
	}

	return true
}


//////////////////
//private func
//////////////////

func (f *Tpl) getTemplateByTag(tag string) *mainTplInfo {
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