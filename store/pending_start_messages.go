package store

import (
	"github.com/cloudfoundry/hm9000/models"
	"reflect"
)

func (store *RealStore) SavePendingStartMessages(messages ...models.PendingStartMessage) error {
	return store.save(messages, store.SchemaRoot()+"/start", 0)
}

func (store *RealStore) GetPendingStartMessages() (map[string]models.PendingStartMessage, error) {
	slice, err := store.get(store.SchemaRoot()+"/start", reflect.TypeOf(map[string]models.PendingStartMessage{}), reflect.ValueOf(models.NewPendingStartMessageFromJSON))
	return slice.Interface().(map[string]models.PendingStartMessage), err
}

func (store *RealStore) DeletePendingStartMessages(messages ...models.PendingStartMessage) error {
	return store.delete(messages, store.SchemaRoot()+"/start")
}
