package utils

import (
	"github.com/widaT/golib/httplib"
	"github.com/widaT/golib/logger"
	"strings"
	"errors"
	"path"
	"time"
	"os"
)
var log *logger.GxLogger

func init() {
	wd, _ := os.Getwd()
	wd = strings.Replace(wd, "\\", "/", -1)
	log = logger.NewLogger(`{"filename":"` + path.Join(wd, "log") + `/etcd.log"}`)
}

type Face struct {
	Feature  []float32         `json:"feature"`
}

type WsResult struct {
	Hash    string `json:"hash"`
	Type    string `json:"type"`
	Faces   []Face `json:"faces"`
	ErrCode int    `json:"err_code,omitempty"`
}

type FResult struct {
	Faces   []Face `json:"faces"`
	ErrCode int    `json:"err_code,omitempty"`
	Type string `json:"type，omitempty"`
}

func GetFeature(url,imgbase64 string) ([]float32, error) {
	//imgbase64 = "data:image/jpeg;base64," + imgbase64 新版接口不需要
	fr  := FResult{}
	req := httplib.Post(url)
	req.Debug(Conf.GetBool("http","debug"))
	timeout := time.Duration(Conf.GetInt("http","timeout"))
	req.SetTimeout(timeout*time.Second,timeout*time.Second)
	req.JSONBody(map[string]string{"data_url":imgbase64})
	err := req.ToJson(&fr)
	if err != nil {
		return nil,err
	}
	if fr.Faces == nil {
		return nil, errors.New("特征提取失败")
	}
	if len(fr.Faces) == 0 {
		return nil, errors.New("没有检测到人脸")
	}
	return fr.Faces[0].Feature,nil
}

func Detect(url,imgbase64 string)(map[string]interface{}, error)  {
	//imgbase64 = "data:image/jpeg;base64," + imgbase64  新版接口不需要
	req := httplib.Post(url)
	req.Debug(Conf.GetBool("http","debug"))
	req.JSONBody(map[string]string{"data_url":imgbase64})
	ret := make(map[string]interface{})
	timeout := time.Duration(Conf.GetInt("http","timeout"))
	req.SetTimeout(timeout*time.Second,timeout*time.Second)
	err := req.ToJson(&ret)
	if err != nil {
		return nil,err
	}
	return ret,nil
}