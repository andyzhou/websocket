package json

import "github.com/andyzhou/websocket/json"

//json info
type GenOptJson struct {
	ErrCode int `json:"errCode"`
	ErrMsg string `json:"errMsg"`
	Kind string `json:"kind"`
	JsonObj interface{} `json:"jsonObj"` //result json object
	json.BaseJson
}

//////////////////////
//api for GenOptRetJson
//////////////////////

//construct
func NewGenOptJson() *GenOptJson {
	this := &GenOptJson{
	}
	return this
}

//encode json data
func (j *GenOptJson) Encode() []byte {
	return j.BaseJson.Encode(j)
}

//decode json data
func (j *GenOptJson) Decode(data []byte) bool {
	return j.BaseJson.Decode(data, j)
}