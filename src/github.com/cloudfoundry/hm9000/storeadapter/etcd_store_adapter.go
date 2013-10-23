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

func NewETCDStoreAdapter(urls []string, maxConcurrentRequests int) *ETCDStoreAdapter {
	return &ETCDStoreAdapter{
		urls:       urls,
		workerPool: workerpool.NewWorkerPool(maxConcurrentRequests),
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
	responses, err := adapter.client.Get(key)
	if adapter.isTimeoutError(err) {
		return StoreNode{}, ErrorTimeout
	}

	if len(responses) == 0 {
		return StoreNode{}, ErrorKeyNotFound
	}
	if err != nil {
		return StoreNode{}, err
	}

	if len(responses) > 1 || responses[0].Key != key {
		return StoreNode{}, ErrorNodeIsDirectory
	}

	return StoreNode{
		Key:   responses[0].Key,
		Value: []byte(responses[0].Value),
		Dir:   responses[0].Dir,
		TTL:   uint64(responses[0].TTL),
	}, nil
}

func (adapter *ETCDStoreAdapter) ListRecursively(key string) (StoreNode, error) {
	responses, err := adapter.client.Get(key)
	if adapter.isTimeoutError(err) {
		return StoreNode{}, ErrorTimeout
	}

	if adapter.isMissingKeyError(err) {
		return StoreNode{}, ErrorKeyNotFound
	}

	if err != nil {
		return StoreNode{}, err
	}

	if len(responses) == 0 {
		return StoreNode{Key: key, Dir: true}, nil
	}

	if responses[0].Key == key {
		return StoreNode{}, ErrorNodeIsNotDirectory
	}

	childNodes := make([]StoreNode, 0)

	for _, response := range responses {
		if response.Key == "/_etcd" {
			continue
		}

		if response.Dir {
			node, err := adapter.ListRecursively(response.Key)
			if err != nil {
				return StoreNode{}, err
			}
			childNodes = append(childNodes, node)
		} else {
			childNodes = append(childNodes, StoreNode{
				Key:   response.Key,
				Value: []byte(response.Value),
				Dir:   response.Dir,
				TTL:   uint64(response.TTL),
			})
		}
	}

	return StoreNode{Key: key, Dir: true, ChildNodes: childNodes}, nil
}

func (adapter *ETCDStoreAdapter) Delete(key string) error {
	_, err := adapter.client.Delete(key)
	if adapter.isTimeoutError(err) {
		return ErrorTimeout
	}

	if adapter.isMissingKeyError(err) {
		return ErrorKeyNotFound
	}

	return err
}
