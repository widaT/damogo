package utils

import (
	"github.com/widaT/damogo/pkg/structure"
	"go.etcd.io/etcd/clientv3"
	"context"
	"time"
	"strings"
	"os"
	"errors"
)

const (
	KEY_PREFIX          = "damo_facedb_group/"
	GATEWAY_NODE_PREFIX = "damo_gateway_notes/"
	FACEDB_NODE_PREFIX  = "damo_facedb_notes/"
	DEFAUT_HOST_KEY     = "damo_facedb_default_host"
)

var cli *clientv3.Client
var err error

func init() {
	cli, err = clientv3.New(clientv3.Config{
		Endpoints:   Conf.GetArray("etcd", "urls", ","),
		DialTimeout: time.Second * 3,
	})

	if err != nil {
		log.Error("etcd 链接错误" + err.Error())
		os.Exit(-1)
	}

	//读取默认的配置
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	resp, err := cli.Get(ctx, DEFAUT_HOST_KEY)
	cancel()
	if err != nil {
		log.Error("etcd 读取默认的配置错误" + err.Error())
		os.Exit(-1)
	}

	for _, ev := range resp.Kvs {
		defaultHost = string(ev.Value)
	}

	//读取group2host map
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*1)
	resp, err = cli.Get(ctx, KEY_PREFIX, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	cancel()
	if err != nil {
		log.Error("etcd 读取group2host map错误" + err.Error())
		os.Exit(-1)
	}
	for _, ev := range resp.Kvs {
		key := strings.Replace(string(ev.Key), KEY_PREFIX,"",-1)
		arr := strings.Split(key, "--")
		SetGroup(arr[0], arr[1], string(ev.Value))
	}



	//watch group
	go func() {
		rch := cli.Watch(context.Background(), KEY_PREFIX, clientv3.WithPrefix())
		for wresp := range rch {
			for _, ev := range wresp.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					key := strings.Replace(string(ev.Kv.Key), KEY_PREFIX,"",-1)
					arr := strings.Split(key, "--")
					if len(arr) != 2 {
						log.Error("etcd Watch EventTypePut err data key：" + key)
					} else {
						SetGroup(arr[0], arr[1], string(ev.Kv.Value))
						log.Info("SetGroup %s %s %s",arr[0],arr[1],string(ev.Kv.Value))
					}

				case clientv3.EventTypeDelete:
					key := strings.Replace(string(ev.Kv.Key), KEY_PREFIX,"",-1)
					arr := strings.Split(key, "--")
					if len(arr) != 2 {
						log.Error("etcd Watch EventTypeDelete err data key：" + key)
					} else {
						delGroup(arr[0], arr[1])
						log.Info("delGroup %s:%s",arr[0],arr[1])
					}
				}
			}
		}
	}()

	//watch defaut host
	go func() {
		rch := cli.Watch(context.Background(), DEFAUT_HOST_KEY)
		for wresp := range rch {
			for _, ev := range wresp.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					defaultHost = string(ev.Kv.Value)
				default:
					log.Error("etcd 监控defaulthost 获取到异常值 %s %s", ev.Type, ev.Kv.Value)
				}
			}
		}
	}()
}

func EtcdDelGroup(groupId string) bool {
	if groupId == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	//注意需要+"--"
	_, err := cli.Delete(ctx, KEY_PREFIX+groupId+"--", clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Error("etcd EtcdDelGroup error" + err.Error())
		return false
	}
	return true
}

func EtcdSetGroup(groupId, host string, keyRange string) bool {
	if groupId == "" || host == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	_, err := cli.Put(ctx, KEY_PREFIX+groupId+"--"+host, keyRange)
	cancel()
	if err != nil {
		log.Error("etcd EtcdAddGroup error" + err.Error())
		return false
	}
	return true
}

func EtcdDelGroupHost(groupId string, host string) bool {
	if groupId == "" || host == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	_, err := cli.Delete(ctx, KEY_PREFIX+groupId+"--"+host)
	cancel()
	if err != nil {
		log.Error("etcd EtcdDelGroupHost error" + err.Error())
		return false
	}
	return true
}

func EtcdSetDefaultHost(info string) error {
	if info == "" {
		return errors.New("host is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	_, err := cli.Put(ctx, DEFAUT_HOST_KEY, info)
	cancel()
	if err != nil {
		log.Error("etcd EtcdAddGroup error" + err.Error())
		return err
	}
	return nil
}

func EtcdRegisterGatewayNode(node string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	_, err = cli.Put(ctx, GATEWAY_NODE_PREFIX+node, time.Now().Format("2006-01-02 15:04:05"))
	cancel()
	if err != nil {
		log.Error("registerNode error" + err.Error())
		return err
	}
	return nil
}

func EtcdGetGatewayNodes() ([]structure.HostInfo, error) {
	var hosts []structure.HostInfo
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	resp, err := cli.Get(ctx, GATEWAY_NODE_PREFIX, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	cancel()
	if err != nil {
		log.Error("etcd GetWayHost 错误" + err.Error())
		return nil, err
	}
	for _, ev := range resp.Kvs {
		host := structure.HostInfo{Name: strings.Replace(string(ev.Key), GATEWAY_NODE_PREFIX,"",-1), RegisterTime: string(ev.Value)}
		hosts = append(hosts, host)
	}
	return hosts, nil
}

func EtcdGetFacedbNodes() ([]structure.HostInfo, error) {
	var hosts []structure.HostInfo
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	resp, err := cli.Get(ctx, FACEDB_NODE_PREFIX, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	cancel()
	if err != nil {
		log.Error("etcd getfacedb 错误" + err.Error())
		return nil, err
	}
	for _, ev := range resp.Kvs {
		host := structure.HostInfo{Name: strings.Replace(string(ev.Key), FACEDB_NODE_PREFIX,"",-1), RegisterTime: string(ev.Value)}
		hosts = append(hosts, host)
	}
	return hosts, nil
}

func CloseEtcd() {
	cli.Close()
}
