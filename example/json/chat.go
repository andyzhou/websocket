package json

import "github.com/andyzhou/websocket/json"

//json info
type ChatJson struct {
	Sender int64 `json:"sender"`
	SenderNick string `json:"senderNick"`
	Receiver int64 `json:"receiver"` //private chat
	Message string `json:"message"`
	Kind int `json:"kind"` //0:system 1:general 2:private 3:tips
	CreateAt int64 `json:"createAt"`
	json.BaseJson
}

//chat tips
type ChatTipsJson struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
	CastAll bool `json:"castAll"`
	CreateAt int64 `json:"createAt"`
	json.BaseJson
}


/////////////////////////
//api for ChatTipsJson
/////////////////////////

//construct
func NewChatTipsJson() *ChatTipsJson {
	this := &ChatTipsJson{
	}
	return this
}

//encode json data
func (j *ChatTipsJson) Encode() []byte {
	return j.BaseJson.Encode(j)
}

//decode json data
func (j *ChatTipsJson) Decode(data []byte) bool {
	return j.BaseJson.Decode(data, j)
}


/////////////////////////
//api for ChatJson
/////////////////////////

//construct
func NewChatJson() *ChatJson {
	this := &ChatJson{
	}
	return this
}

//encode json data
func (j *ChatJson) Encode() []byte {
	return j.BaseJson.Encode(j)
}

//decode json data
func (j *ChatJson) Decode(data []byte) bool {
	return j.BaseJson.Decode(data, j)
}