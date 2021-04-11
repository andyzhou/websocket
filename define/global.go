package define

//inter request url pattern
const (
	ReqUrlOfPagePattern = "/page/{page}"
	ReqUrlOfFilePattern = "/file/{file}"
	ReqUrlOfHttpReqPattern = "/http/{request}"
	ReqUrlOfChannelPattern = "/channel/{channel}"
)

//inter param
const (
	PageParaName = "page"
	FileParaName = "file"
	HttpReqParaName = "request"
	ChannelParaName = "channel"
)

//router method
const (
	RouterMethodOfGet = "GET"
	RouterMethodOfPost = "POST"
)

//global
const (
	TplFileExtName = ".tpl"
)
