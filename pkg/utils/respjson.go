package utils

import (
	"time"
	"encoding/json"
)

type JsonResp struct {
	ErrorCode int         `json:"error_code,omitempty"`
	ErrorMsg  string      `json:"error_msg,omitempty"`
	ResultNum int         `json:"result_num,omitempty"`
	Result    interface{} `json:"result,omitempty"`
	LogId     int         `json:"log_id"`
}

func NewJsonResp() JsonResp {
	return JsonResp{LogId: time.Now().Second()}
}

func (j *JsonResp) ToJson() string {
	byte, _ := json.Marshal(j)
	return string(byte)
}
