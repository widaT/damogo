package main

import (
	"github.com/widaT/damogo/pkg/utils"
	"github.com/widaT/golib/web/server"
	"github.com/widaT/damogo/pkg/pb"
	"github.com/widaT/golib/logger"
	"github.com/widaT/golib/net2"
	"strconv"
	"context"
	"strings"
	"errors"
	"time"
	"path"
	"os"
	"sort"
	"fmt"
	"sync"
)

const (
	PARAMS_ERROR   = 1001 //参数错误
	FEATURE_ERROR  = 1002 //特征提取失败
	IDENTIFY_ERROR = 1003 //人脸识别rpc服务调用失败
	ADD_USER_ERROR = 1004 //人脸识别rpc服务调用失败
	RPC_ERROR      = 1005 //rpc调用失败
	GROUP_NO_FOUND = 1006 //rpc调用失败
	FAIL           = 400  //失败
)

type UserInfo struct {
	UID        string                 `json:"uid"`
	GroupId    string                 `json:"group_id"`
	Distance   float32   			  `json:"scores"`
}

type ResultInfo struct {
	Distance float64 `json:"-"`
	User     UserInfo
}

type ResultSlice []ResultInfo

func (p ResultSlice) Len() int           { return len(p) }
func (p ResultSlice) Less(i, j int) bool { return p[i].Distance < p[j].Distance }
func (p ResultSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type Arg struct {
	Feature []float32
	GroupId string
}

var (
	log *logger.GxLogger
)

func init() {
	wd, _ := os.Getwd()
	wd = strings.Replace(wd, "\\", "/", -1)
	log = logger.NewLogger(`{"filename":"` + path.Join(wd, "log") + `/log.log"}`)
}
//相似度算法
func similarity(diff float32) float32 {
	threshold :=float32(utils.Conf.GetFloat32("base","threshold"))
	maxDiff := float32(0.6*threshold + threshold)
	similarity := float32(0.0)
	if diff > maxDiff {
		similarity = 0.0
	} else {
		similarity = (maxDiff - diff) / threshold
		if similarity > 1.0 {
			similarity = 1.0
		}
	}
	return similarity * 100
}

//人脸认证
func recognition(ctx *server.Context) string {
	ctx.ContentType("application/json")
	jsonResp := utils.NewJsonResp()
	groupId := strings.TrimSpace(ctx.Params["group_id"])
	image := strings.TrimSpace(ctx.Params["image"])
	if groupId == "" || image == "" {
		jsonResp.ErrorCode = PARAMS_ERROR
		jsonResp.ErrorMsg = "参数错误"
		return jsonResp.ToJson()
	}
	if !utils.GroupExist(groupId) {
		jsonResp.ErrorCode = GROUP_NO_FOUND
		jsonResp.ErrorMsg = "分组不存在"
		return jsonResp.ToJson()
	}

	features, err := getFeatures(image)
	if err != nil {
		jsonResp.ErrorCode = FEATURE_ERROR
		jsonResp.ErrorMsg = "特征提取失败:" + err.Error()
		log.Error(jsonResp.ErrorMsg)
		return jsonResp.ToJson()
	}

	arg := Arg{features, groupId}
	stime := time.Now()
	ret, err := identify(arg, groupId)
	if err != nil {
		jsonResp.ErrorCode = IDENTIFY_ERROR
		jsonResp.ErrorMsg = "人脸识别rpc服务调用失败:" + err.Error()
		log.Error(jsonResp.ErrorMsg)
		return jsonResp.ToJson()
	}
	log.Info("人脸匹配使用了 %d ms", time.Now().Sub(stime).Nanoseconds()/1000000)
	var respinfo []UserInfo
	for _, info := range *ret {
		var user UserInfo
		user.UID = info.User.UID
		user.GroupId = info.User.GroupId
		confidence := similarity(info.User.Distance )
		user.Distance = confidence
		respinfo = append(respinfo, user)
	}
	jsonResp.Result = respinfo
	jsonResp.ResultNum = len(*ret)
	return jsonResp.ToJson()
}


func getFeatures(image string) ([]float32, error) {
	url := utils.Conf.GetString("http", "feature")
	startTime := time.Now()
	features, err := utils.GetFeature(url, image)
	log.Info("call getFeatures use %d ms", time.Now().Sub(startTime).Nanoseconds()/1000000)
	if err != nil {
		log.Error("特征提取失败:" + err.Error())
		return nil, err
	}
	if len(features) == 0 || len(features) != 512 {
		log.Error("人脸没有识别到人脸或者feature不是512维")
		return nil, errors.New("face is not detected or feature length not match")
	}
	return features, nil
}


func identify(arg Arg, groupId string) (reply *[]ResultInfo, err error) {
	conns, err := utils.GetRpcClientsByGroup(groupId)
	if err != nil {
		return nil, errors.New("链接rpc服务器失败:" + err.Error())
	}
	defer utils.CloseClients(conns)
	var ret []ResultInfo
	length := len(conns)
	retTemp := make([]*pb.SearchReply, length)
	params := &pb.SearchRequest{Group:groupId,Feature:arg.Feature}
	wg := sync.WaitGroup{}
	//并发请求 @todo 需要优化
	var gerr error = nil
	for i, conn := range conns {
		wg.Add(1)
		go func (i int){
			client:= pb.NewFacedbClient(conn)
			retTemp[i],err = client.Search(context.Background(), params)
			if err != nil {
				gerr = err
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	if gerr != nil {
		return nil,gerr
	}
	//汇总数据
	for i := 0; i < length; i++ {
		for  _,a := range retTemp[i].Users{
			ret = append(ret,ResultInfo{Distance:float64(a.Distance),User:UserInfo{UID:a.Name}})
		}
	}
	//为了满足自动分裂时可能出现的一个uid分布在不同的region的可能，结果需要排序和去重
	sort.Sort(ResultSlice(ret))
	tempMap := make(map[string]bool)
	var top5 []ResultInfo
	i := 0
	for _, user := range ret {
		if _, found := tempMap[user.User.UID]; !found {
			i++
			top5 = append(top5, user)
			if i == 5 {
				break
			}
		}
	}
	reply = &top5
	return
}

//用户添加
func useAdd(ctx *server.Context) string {
	ctx.ContentType("application/json")
	jsonResp := utils.NewJsonResp()
	uid := strings.TrimSpace(ctx.Params["uid"])
	groupId := strings.TrimSpace(ctx.Params["group_id"])
	image := strings.TrimSpace(ctx.Params["image"])
	actionType := strings.TrimSpace(ctx.Params["action_type"]) //暂时该参数无效
	if actionType == "" || actionType == "append" {
		actionType = "append"
	} else if actionType == "replace" {
		actionType = "replace"
	} else {
		jsonResp.ErrorCode = PARAMS_ERROR
		jsonResp.ErrorMsg = "参数错误"
		return jsonResp.ToJson()
	}
	if uid == "" || groupId == "" || image == "" {
		jsonResp.ErrorCode = PARAMS_ERROR
		jsonResp.ErrorMsg = "参数错误"
		return jsonResp.ToJson()
	}
	features, err := getFeatures(image)
	if err != nil {
		jsonResp.ErrorCode = FEATURE_ERROR
		jsonResp.ErrorMsg = "人脸特征提取失败:" + err.Error()
		log.Error(jsonResp.ErrorMsg)
		return jsonResp.ToJson()
	}
	userInfo := pb.UserInfo{Group:groupId,Id:uid,Feature:features}
	conn, err := utils.GetRpcClientByUid(groupId, uid, true)
	if err != nil {
		jsonResp.ErrorCode = ADD_USER_ERROR
		jsonResp.ErrorMsg = "添加用户rpc服务调用失败:" + err.Error()
		log.Error(jsonResp.ErrorMsg)
		return jsonResp.ToJson()
	}
	defer conn.Close()
	client := pb.NewFacedbClient(conn)
	ret ,err := client.AddUser(context.Background(), &userInfo)
	if err != nil {
		jsonResp.ErrorCode = ADD_USER_ERROR
		jsonResp.ErrorMsg = "添加用户rpc服务调用失败:" + err.Error()
		log.Error(jsonResp.ErrorMsg)
		return jsonResp.ToJson()
	}
	if ret.Ret {
		return jsonResp.ToJson()
	}
	jsonResp.ErrorMsg = "添加失败"
	jsonResp.ErrorCode = FAIL
	return jsonResp.ToJson()
}

func delGroup(ctx *server.Context) string {
	ctx.ContentType("application/json")
	jsonResp := utils.NewJsonResp()
	groupId := strings.TrimSpace(ctx.Params["group_id"])
	if groupId == "" {
		jsonResp.ErrorCode = PARAMS_ERROR
		jsonResp.ErrorMsg = "参数错误"
		return jsonResp.ToJson()
	}

	if !utils.GroupExist(groupId) {
		jsonResp.ErrorCode = GROUP_NO_FOUND
		jsonResp.ErrorMsg = "分组不存在"
		return jsonResp.ToJson()
	}

	conns, err := utils.GetRpcClientsByGroup(groupId)
	if err != nil {
		log.Error("链接rpc服务器失败:" + err.Error())
		jsonResp.ErrorCode = RPC_ERROR
		jsonResp.ErrorMsg = "rpc调用出错"
		return jsonResp.ToJson()
	}
	defer utils.CloseClients(conns)

	length := len(conns)
	retTemp := make([]*pb.NomalReply, length)
	//并发请求
	var gerr error = nil
	wg:= sync.WaitGroup{}
	for i, conn := range conns {
		wg.Add(1)
		go func(i int){
			client := pb.NewFacedbClient(conn)
			retTemp[i],err = client.DelGroup(context.Background(), &pb.Group{Group:groupId})
			if err!= nil {
				gerr = err
			}
			wg.Done()
		}(i)
	}

	//retTemp 非顺序赋值
	j := 0
	for _, r := range retTemp {
		if r.Ret {
			j ++
		}
	}
	if err != nil {
		log.Error("删除分组失败：" + err.Error())
		jsonResp.ErrorCode = RPC_ERROR
		jsonResp.ErrorMsg = "rpc调用出错"
		return jsonResp.ToJson()
	}
	if j != length {
		log.Error("删除分组失败：" + err.Error())
		jsonResp.ErrorMsg = "删除分组失败"
		jsonResp.ErrorCode = FAIL
	}
	//删除缓存 同时删除etcd配置 同步到其他节点
	utils.DelGroup(groupId)
	return jsonResp.ToJson()
}

func delUser(ctx *server.Context) string {
	ctx.ContentType("application/json")
	jsonResp := utils.NewJsonResp()
	uid := strings.TrimSpace(ctx.Params["uid"])
	groupId := strings.TrimSpace(ctx.Params["group_id"])
	if uid == "" || groupId == "" {
		jsonResp.ErrorCode = PARAMS_ERROR
		jsonResp.ErrorMsg = "参数错误"
		return jsonResp.ToJson()
	}
	if !utils.GroupExist(groupId) {
		jsonResp.ErrorCode = GROUP_NO_FOUND
		jsonResp.ErrorMsg = "分组不存在"
		return jsonResp.ToJson()
	}
	conn, err := utils.GetRpcClientByUid(groupId, uid, false)
	if err != nil {
		log.Error("GetRpcClientByUid error:" + err.Error())
		jsonResp.ErrorCode = RPC_ERROR
		jsonResp.ErrorMsg = "rpc调用出错"
		return jsonResp.ToJson()
	}
	defer conn.Close()
	userInfo := pb.UserInfo{Group: groupId,Id:uid}

	client := pb.NewFacedbClient(conn)
	ret, err := client.DelUser(context.Background(),&userInfo)
	if !ret.Ret {
		jsonResp.ErrorMsg = "删除用户失败"
		jsonResp.ErrorCode = FAIL
	}
	return jsonResp.ToJson()
}

func groupList(ctx *server.Context) string {
	ctx.ContentType("application/json")
	jsonResp := utils.NewJsonResp()
	jsonResp.Result = utils.GetGroup()
	return jsonResp.ToJson()
}

func userList(ctx *server.Context) string {
	ctx.ContentType("application/json")
	jsonResp := utils.NewJsonResp()
	groupId := strings.TrimSpace(ctx.Params["group_id"])
	startKey := strings.TrimSpace(ctx.Params["start_key"])
	num, _ := strconv.Atoi(strings.TrimSpace(ctx.Params["num"]))
	if groupId == "" {
		jsonResp.ErrorCode = PARAMS_ERROR
		jsonResp.ErrorMsg = "参数错误"
		return jsonResp.ToJson()
	}

	if !utils.GroupExist(groupId) {
		jsonResp.ErrorCode = GROUP_NO_FOUND
		jsonResp.ErrorMsg = "分组不存在"
		return jsonResp.ToJson()
	}

	if startKey == "" {
		startKey = "0"
	}
	//num值默认为1000
	if num == 0 {
		num = utils.Conf.GetInt("base", "user_list_num")
	}
	var ret []string
	var i = 0 //死循环保险
	for {
		i++
		endKey := ""
		conn, err := utils.GetRpcClientUserRange(groupId, startKey, &endKey)
		if err != nil {
			if err == utils.OVERRANGE {
				log.Error("GetRpcClientUserRange err:" + err.Error())
				break
			}
			log.Error("链接rpc服务器失败:" + err.Error())
			jsonResp.ErrorCode = RPC_ERROR
			jsonResp.ErrorMsg = "rpc调用出错"
			return jsonResp.ToJson()
		}

		arg := pb.UserListReq{Group: groupId, Skey: startKey,Num: int32(num - len(ret))}

		client := pb.NewFacedbClient(conn)
		users,err := client.UserList(context.Background(),&arg)
		if err != nil {
			log.Error("链接rpc服务器失败:" + err.Error())
			jsonResp.ErrorCode = RPC_ERROR
			jsonResp.ErrorMsg = "rpc调用出错"
			conn.Close()
			return jsonResp.ToJson()
		}
		ret = append(ret, users.Values...)
		conn.Close()
		if i > 10 { //最大遍历10个机器
			break
		}
		if len(ret) < num && endKey != "" {
			startKey = endKey
			continue
		} else {
			break
		}
	}
	jsonResp.Result = ret
	return jsonResp.ToJson()
}

func getuser(ctx *server.Context) string {
	ctx.ContentType("application/json")
	jsonResp := utils.NewJsonResp()
	uid := strings.TrimSpace(ctx.Params["uid"])
	groupId := strings.TrimSpace(ctx.Params["group_id"])
	if uid == "" || groupId == "" {
		jsonResp.ErrorCode = PARAMS_ERROR
		jsonResp.ErrorMsg = "参数错误"
		return jsonResp.ToJson()
	}

	if !utils.GroupExist(groupId) {
		jsonResp.ErrorCode = GROUP_NO_FOUND
		jsonResp.ErrorMsg = "分组不存在"
		return jsonResp.ToJson()
	}

	conn, err := utils.GetRpcClientByUid(groupId, uid, false)
	if err != nil {
		log.Error("链接rpc服务器失败:" + err.Error())
		jsonResp.ErrorCode = RPC_ERROR
		jsonResp.ErrorMsg = "rpc调用出错"
		return jsonResp.ToJson()
	}
	defer conn.Close()
	userInfo := pb.UserInfo{Group: groupId,Id:uid}
	client := pb.NewFacedbClient(conn)
	user,err := client.GetUser(context.Background(),&userInfo)
	if err != nil {
		jsonResp.ErrorMsg = "获取用户失败"
		jsonResp.ErrorCode = FAIL
		return jsonResp.ToJson()
	}

	jsonResp.Result = user
	return jsonResp.ToJson()
}


func detect(ctx *server.Context) string {
	ctx.ContentType("application/json")
	image := strings.TrimSpace(ctx.Params["image"])
	jsonResp := utils.NewJsonResp()
	if image == "" {
		jsonResp.ErrorCode = PARAMS_ERROR
		jsonResp.ErrorMsg = "参数错误"
		return jsonResp.ToJson()
	}
	url := utils.Conf.GetString("http", "detect")
	stime := time.Now()
	ret, err := utils.Detect(url,image)
	log.Info("call getFeatures use %d ms", time.Now().Sub(stime).Nanoseconds()/1000000)
	if err != nil {
		jsonResp.ErrorCode = RPC_ERROR
		jsonResp.ErrorMsg = "接口调用错误:" + err.Error()
		return jsonResp.ToJson()
	}
	jsonResp.Result = ret
	return jsonResp.ToJson()
}

func main() {
	defer utils.CloseEtcd()
	localIp := net2.GetLocalIP()
	port := utils.Conf.GetString("base", "port")
	node := fmt.Sprintf("%s:%s", localIp, port)
	//注册facedb节点
	if err := utils.EtcdRegisterGatewayNode(node); err != nil {
		log.Error("EtcdRegisterGatewayNode err:" + err.Error())
		return
	}
	server.AddRoute("/", recognition)
	server.AddRoute("/identify", recognition)
	server.AddRoute("/adduser", useAdd)
	server.AddRoute("/deluser", delUser)
	server.AddRoute("/delgroup", delGroup)
	server.AddRoute("/grouplist", groupList)
	server.AddRoute("/userlist", userList)
	server.AddRoute("/detect", detect)
	server.AddRoute("/getuser", getuser)
	server.Run(":" + utils.Conf.GetString("base", "port"))
}

