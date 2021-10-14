package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	servicesPrefix = "/micro/registry"
	metricsPrefix  = ".metrics"
	pathPat        = regexp.MustCompile("/micro/registry")
	etcdServer     = flag.String("server", "etcd:2379", "etcd server to connect to")
	targetFile     = flag.String("target-file", "sd_etcd.json", "the file that contains the target groups")
)

type (
	instance struct {
		Name      string      `json:"name"`
		Version   string      `json:"version"`
		Metadata  interface{} `json:"metadata"`
		Endpoints interface{} `json:"endpoints"`
		Nodes     []node      `json:"nodes"`
	}
	node struct {
		Id      string `json:"id"`
		Address string `json:"address"`
	}
	sd struct {
		Targets []string          `json:"targets"`
		Labels  map[string]string `json:"labels"`
	}
	services map[string]sd
)

func UnmarshalInstance(data []byte) (instance, error) {
	var r instance
	err := json.Unmarshal(data, &r)
	r.Name = strings.ReplaceAll(r.Name, metricsPrefix, "")
	return r, err
}

func (srvs services) handle(handler func(*mvccpb.KeyValue), kvs ...*mvccpb.KeyValue) {
	for _, kv := range kvs {
		if pathPat.MatchString(string(kv.Key)) && strings.Contains(string(kv.Key), metricsPrefix) {
			handler(kv)
		}
	}
}

func (srvs services) update(node *mvccpb.KeyValue) {
	i, err := UnmarshalInstance(node.Value)
	if err != nil {
		log.Print(err)
		return
	}
	_, ok := srvs[string(node.Key)]
	if !ok {
		srvs[string(node.Key)] = sd{Labels: map[string]string{}, Targets: []string{}}
	}
	sdd := srvs[string(node.Key)]
	for _, node := range i.Nodes {
		sdd.Targets = append(sdd.Targets, node.Address)
	}
	sdd.Labels["name"] = i.Name
	sdd.Labels["version"] = i.Version
	srvs[string(node.Key)] = sdd
}

func (srvs services) delete(node *mvccpb.KeyValue) {
	delete(srvs, string(node.Key))
}

type TargetGroup struct {
	Targets []string          `json:"targets,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}

func (srvs services) persist() {
	var items []sd
	for _, instances := range srvs {
		items = append(items, instances)
	}
	content, err := json.Marshal(items)
	if err != nil {
		fmt.Println("persist#Marshal#", err)
		return
	}

	if _, err := os.Stat(filepath.Dir(*targetFile)); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(*targetFile), 0700)
	}

	f, err := os.Create(*targetFile)
	if err != nil {
		fmt.Println("persist#Create#", err)
		return
	}
	defer f.Close()
	if _, err := f.Write(content); err != nil {
		log.Print(err)
	}
}

func main() {
	flag.Parse()
	ctx := context.Background()
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{*etcdServer},
		DialTimeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatalf("Error on initial retrieval: %s", err)
	}
	res, err := client.Get(ctx, servicesPrefix, clientv3.WithPrefix())
	if err != nil {
		log.Fatalf("Error on initial retrieval: %s", err)
	}
	var srvs = services{}
	srvs.handle(srvs.update, res.Kvs...)
	srvs.persist()

	watcher := clientv3.NewWatcher(client)
	watchChan := watcher.Watch(ctx, servicesPrefix, clientv3.WithPrefix())
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			switch event.Type {
			case mvccpb.PUT:
				srvs.handle(srvs.update, event.Kv)
			case mvccpb.DELETE:
				srvs.handle(srvs.delete, event.Kv)
			}
			srvs.persist()
		}
	}
}
