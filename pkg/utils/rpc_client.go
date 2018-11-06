package utils

import (
	"sync"
	"strings"
	"errors"
	"google.golang.org/grpc"
)

func GetFacedb(address string)(*grpc.ClientConn,error)  {
	return  grpc.Dial(address, grpc.WithInsecure())
}

var OVERRANGE = errors.New("uid over the range")

type Region struct {
	StartKey string `json:"start_key"`
	EndKey   string `json:"end_key"`
}

var Group2ServerMap = make(map[string]map[string]Region)
var mut sync.RWMutex
var defaultHost string // 默认机器存储在etcd配置里头  ep "localhost:8002||0||{"

func GetGroup() []string {
	mut.RLock()
	defer mut.RUnlock()
	var groups []string
	for group := range Group2ServerMap {
		groups = append(groups, group)
	}
	return groups
}


func SetGroup(groupId, host, keyRange string) {
	mut.Lock()
	defer mut.Unlock()
	arr := strings.Split(keyRange, "||")
	_, found := Group2ServerMap[groupId]
	if found {
		Group2ServerMap[groupId][host] = Region{arr[0], arr[1]}
	} else {
		Group2ServerMap[groupId] = make(map[string]Region)
		Group2ServerMap[groupId][host] = Region{arr[0], arr[1]}
	}
}

func GetHosts() []string {
	mut.RLock()
	defer mut.RUnlock()
	var tmp = make(map[string]bool)
	var hosts []string
	for _, v := range Group2ServerMap {
		for host := range v {
			tmp[host] = true
		}
	}
	for k := range tmp {
		hosts = append(hosts, k)
	}
	return hosts
}

func GroupExist(groupId string) bool {
	mut.RLock()
	defer mut.RUnlock()
	_, found := Group2ServerMap[groupId]
	return found
}

func DelGroup(groupId string) {
	//使用etcd 同步删除 group
	EtcdDelGroup(groupId)
}

//删除group的host节点或者删除group
func delGroup(groupId, host string) {
	mut.Lock()
	defer mut.Unlock()
	_, found := Group2ServerMap[groupId]
	if found {
		delete(Group2ServerMap[groupId], host)
		//删除group
		if len(Group2ServerMap[groupId]) == 0 {
			delete(Group2ServerMap,groupId)
		}
	}
}

func GetRpcClientByUid(groupId, uid string, needDefautHost bool) (*grpc.ClientConn,error) {
	mut.RLock()
	defer mut.RUnlock()
	if _, found := Group2ServerMap[groupId]; !found || len(Group2ServerMap[groupId]) == 0 {
		//获取默认机器
		if needDefautHost {
			host := strings.Split(defaultHost, "||")
			conn,err := GetFacedb (host[0])
			if err != nil {
				log.Error("group %s get defaulthost %s connect err %s",groupId,host[0],err.Error())
				return nil,err
			}
			EtcdSetGroup(groupId, host[0], host[1]+"||"+host[2])
			return conn,nil
		}
		return nil, errors.New("host not found")
	}
	for host, region := range Group2ServerMap[groupId] {
		//字节比较 左闭右闭
		if uid >= region.StartKey && uid <= region.EndKey {
			return GetFacedb(host)
		}
	}
	return nil, errors.New("not host in the uid regions")
}

func GetRpcClientUserRange(groupId, uid string, endKey *string) (*grpc.ClientConn, error) {
	mut.RLock()
	defer mut.RUnlock()
	if _, found := Group2ServerMap[groupId]; !found || len(Group2ServerMap[groupId]) == 0 {
		//获取默认机器
		return nil, errors.New("host not found")
	}
	for host, region := range Group2ServerMap[groupId] {
		//字节比较 左闭右闭
		if uid >= region.StartKey && uid <= region.EndKey {
			*endKey = region.EndKey
			return GetFacedb(host)
		}
	}
	return nil, OVERRANGE
}

func GetRpcClientsByGroup(groupId string) ([]*grpc.ClientConn, error) {
	mut.RLock()
	defer mut.RUnlock()
	var clients []*grpc.ClientConn
	if _, found := Group2ServerMap[groupId]; !found || len(Group2ServerMap[groupId]) == 0 {
		return nil, errors.New("host not found")
	}
	for host := range Group2ServerMap[groupId] {
		client, err := GetFacedb(host)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	if len(clients) == 0 {
		return nil, errors.New("not host in the uid regions")
	}else {
		return clients, nil
	}
}

func CloseClients(clients []*grpc.ClientConn) {
	for _, c := range clients {
		c.Close()
	}
}
