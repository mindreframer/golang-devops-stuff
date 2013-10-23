package hm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"sort"
	"time"
)

func Dump(l logger.Logger, conf config.Config) {
	etcdStoreAdapter := connectToETCDStoreAdapter(l, conf)
	fmt.Printf("Dump - Current timestamp %d\n", time.Now().Unix())

	entries := sort.StringSlice{}

	node, err := etcdStoreAdapter.ListRecursively("/")
	if err != nil {
		panic(err)
	}
	walk(node, func(node storeadapter.StoreNode) {
		ttl := fmt.Sprintf("[TTL:%ds]", node.TTL)
		if node.TTL == 0 {
			ttl = "[TTL: ∞]"
		}
		buf := &bytes.Buffer{}
		err := json.Indent(buf, node.Value, "    ", "  ")
		value := buf.String()
		if err != nil {
			value = string(node.Value)
		}
		entries = append(entries, fmt.Sprintf("%s %s:\n    %s", node.Key, ttl, value))
	})

	sort.Sort(entries)
	for _, entry := range entries {
		fmt.Printf(entry + "\n")
	}
}

func Clear(l logger.Logger, conf config.Config) {
	etcdStoreAdapter := connectToETCDStoreAdapter(l, conf)
	l.Info(fmt.Sprintf("Clear - Current timestamp %d\n", time.Now().Unix()))

	node, err := etcdStoreAdapter.ListRecursively("/")
	if err != nil {
		panic(err)
	}
	walk(node, func(node storeadapter.StoreNode) {
		etcdStoreAdapter.Delete(node.Key)
	})
}

func walk(node storeadapter.StoreNode, callback func(storeadapter.StoreNode)) {
	for _, node := range node.ChildNodes {
		if node.Dir {
			walk(node, callback)
		} else {
			callback(node)
		}
	}
}
