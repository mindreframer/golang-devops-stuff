package store

import (
	"github.com/cloudfoundry/storeadapter"
	"strconv"
)

func (store *RealStore) SaveMetric(metric string, value float64) error {
	node := storeadapter.StoreNode{
		Key:   store.SchemaRoot() + "/metrics/" + metric,
		Value: []byte(strconv.FormatFloat(value, 'f', 5, 64)),
	}
	return store.adapter.SetMulti([]storeadapter.StoreNode{node})
}

func (store *RealStore) GetMetric(metric string) (float64, error) {
	node, err := store.adapter.Get(store.SchemaRoot() + "/metrics/" + metric)
	if err != nil {
		return -1, err
	}

	return strconv.ParseFloat(string(node.Value), 64)
}
