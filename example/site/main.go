package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

//inter macro define
const (
	WebPort = 8090
)
const (
	WebTpl  = "web/tpl/*.html"
	WebHtml = "web/html"
)

const (
	UriOfRoot = "/"
	UriOfHtml = "/html"
)

const (
	ParaOfAct      = "act"
)

// tpl
const (
	TplOfHome = "home.html"
)

//global variable
var (
	g *gin.Engine
)

//init gin engine
func initGin() {
	gin.SetMode(gin.ReleaseMode)
	g = gin.Default()
	g.Use(gin.Recovery())

	//get app run dir
	curDir, _ := os.Getwd()

	//setup tpl and static path
	webTplDir := fmt.Sprintf("%v/%v", curDir, WebTpl)
	webStaticDir := fmt.Sprintf("%v/%v", curDir, WebHtml)

	//init templates
	g.LoadHTMLGlob(webTplDir)

	//init static path
	g.Static(UriOfHtml, webStaticDir)

	//register request entry
	g.Any(UriOfRoot, webEntry)
}

//init
func init()  {
	initGin()
}

//web entry
func webEntry(ctx *gin.Context) {
	//get act
	act, _ := ctx.GetQuery(ParaOfAct)
	switch act {
	default:
		showHomePage(ctx)
	}
}

//show home page
func showHomePage(ctx *gin.Context) {
	//out put page
	ctx.HTML(http.StatusOK, TplOfHome, nil)
}

//main func
func main() {
	var (
		wg sync.WaitGroup
	)
	//wg add
	wg.Add(1)

	//start gin service
	serverAddr := fmt.Sprintf(":%v", WebPort)
	go g.Run(serverAddr)
	fmt.Printf("web start at port %v\n", WebPort)

	wg.Wait()
	fmt.Printf("web stopped!\n")
}


