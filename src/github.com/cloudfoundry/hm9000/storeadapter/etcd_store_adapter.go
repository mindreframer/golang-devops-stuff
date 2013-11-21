package storeadapter

import (
	"github.com/cloudfoundry/hm9000/helpers/workerpool"
	"github.com/coreos/go-etcd/etcd"
)

type ETCDStoreAdapter struct {
	urls       []string
	client     *etcd.Client
	workerPool *workerpool.WorkerPool
}

func NewETCDStoreAdapter(urls []string, workerPool *workerpool.WorkerPool) *ETCDStoreAdapter {
	return &ETCDStoreAdapter{
		urls:       urls,
		workerPool: workerPool,
	}
}

func (adapter *ETCDStoreAdapter) Connect() error {
	adapter.client = etcd.NewClient(adapter.urls)

	return nil
}

func (adapter *ETCDStoreAdapter) Disconnect() error {
	adapter.workerPool.StopWorkers()

	return nil
}

func (adapter *ETCDStoreAdapter) isTimeoutError(err error) bool {
	return err != nil && err.Error() == "Cannot reach servers"
}

func (adapter *ETCDStoreAdapter) isMissingKeyError(err error) bool {
	if err != nil {
		etcdError, ok := err.(etcd.EtcdError)
		if ok && etcdError.ErrorCode == 100 { //yup.  100.
			return true
		}
	}
	return false
}

func (adapter *ETCDStoreAdapter) isNotAFileError(err error) bool {
	if err != nil {
		etcdError, ok := err.(etcd.EtcdError)
		if ok && etcdError.ErrorCode == 102 { //yup.  102.
			return true
		}
	}
	return false
}

func (adapter *ETCDStoreAdapter) Set(nodes []StoreNode) error {
	results := make(chan error, len(nodes))

	for _, node := range nodes {
		node := node
		adapter.workerPool.ScheduleWork(func() {
			_, err := adapter.client.Set(node.Key, string(node.Value), node.TTL)
			results <- err
		})
	}

	var err error
	numReceived := 0
	for numReceived < len(nodes) {
		result := <-results
		numReceived++
		if err == nil {
			err = result
		}
	}

	if adapter.isNotAFileError(err) {
		return ErrorNodeIsDirectory
	}

	if adapter.isTimeoutError(err) {
		return ErrorTimeout
	}

	return err
}

func (adapter *ETCDStoreAdapter) Get(key string) (StoreNode, error) {
	//TODO: remove this terribleness when we upgrade go-etcd
	if key == "/" {
		key = "/?garbage=foo&"
	}

	done := make(chan bool, 1)
	var response *etcd.Response
	var err error

	//we route through the worker pool to enable usage tracking
	adapter.workerPool.ScheduleWork(func() {
		response, err = adapter.client.Get(key, false)
		done <- true
	})

	<-done

	if adapter.isTimeoutError(err) {
		return StoreNode{}, ErrorTimeout
	}

	if adapter.isMissingKeyError(err) {
		return StoreNode{}, ErrorKeyNotFound
	}
	if err != nil {
		return StoreNode{}, err
	}

	if response.Dir {
		return StoreNode{}, ErrorNodeIsDirectory
	}

	return StoreNode{
		Key:   response.Key,
		Value: []byte(response.Value),
		Dir:   response.Dir,
		TTL:   uint64(response.TTL),
	}, nil
}

func (adapter *ETCDStoreAdapter) ListRecursively(key string) (StoreNode, error) {
	//TODO: remove this terribleness when we upgrade go-etcd
	if key == "/" {
		key = "/?recursive=true&garbage=foo&"
	}

	done := make(chan bool, 1)
	var response *etcd.Response
	var err error

	//we route through the worker pool to enable usage tracking
	adapter.workerPool.ScheduleWork(func() {
		response, err = adapter.client.GetAll(key, false)
		done <- true
	})

	<-done

	if adapter.isTimeoutError(err) {
		return StoreNode{}, ErrorTimeout
	}

	if adapter.isMissingKeyError(err) {
		return StoreNode{}, ErrorKeyNotFound
	}

	if err != nil {
		return StoreNode{}, err
	}

	if !response.Dir {
		return StoreNode{}, ErrorNodeIsNotDirectory
	}

	if len(response.Kvs) == 0 {
		return StoreNode{Key: key, Dir: true}, nil
	}

	kvPair := etcd.KeyValuePair{
		Key:     response.Key,
		Value:   response.Value,
		Dir:     response.Dir,
		KVPairs: response.Kvs,
	}

	return adapter.makeStoreNode(kvPair), nil
}

func (adapter *ETCDStoreAdapter) makeStoreNode(kvPair etcd.KeyValuePair) StoreNode {
	if kvPair.Dir {
		node := StoreNode{
			Key:        kvPair.Key,
			Dir:        true,
			ChildNodes: []StoreNode{},
		}

		for _, child := range kvPair.KVPairs {
			node.ChildNodes = append(node.ChildNodes, adapter.makeStoreNode(child))
		}

		return node
	} else {
		return StoreNode{
			Key:   kvPair.Key,
			Value: []byte(kvPair.Value),
			// TTL:   uint64(kvPair.TTL),
		}
	}
}

func (adapter *ETCDStoreAdapter) Delete(keys ...string) error {
	results := make(chan error, len(keys))

	for _, key := range keys {
		key := key
		adapter.workerPool.ScheduleWork(func() {
			_, err := adapter.client.DeleteAll(key)
			results <- err
		})
	}

	var err error
	numReceived := 0
	for numReceived < len(keys) {
		result := <-results
		numReceived++
		if err == nil {
			err = result
		}
	}

	if adapter.isTimeoutError(err) {
		return ErrorTimeout
	}

	if adapter.isMissingKeyError(err) {
		return ErrorKeyNotFound
	}

	return err
}
