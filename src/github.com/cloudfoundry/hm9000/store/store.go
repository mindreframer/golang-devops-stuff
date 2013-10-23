package store

import (
	"errors"
	"fmt"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"reflect"
	"time"
)

var ActualIsNotFreshError = errors.New("Actual state is not fresh")
var DesiredIsNotFreshError = errors.New("Desired state is not fresh")
var ActualAndDesiredAreNotFreshError = errors.New("Actual and desired state are not fresh")
var AppNotFoundError = errors.New("App not found")

type Storeable interface {
	StoreKey() string
	ToJSON() []byte
}

type Store interface {
	BumpDesiredFreshness(timestamp time.Time) error
	BumpActualFreshness(timestamp time.Time) error

	IsDesiredStateFresh() (bool, error)
	IsActualStateFresh(time.Time) (bool, error)

	VerifyFreshness(time.Time) error

	AppKey(appGuid string, appVersion string) string
	GetApps() (map[string]*models.App, error)
	GetApp(appGuid string, appVersion string) (*models.App, error)

	SaveDesiredState(desiredStates ...models.DesiredAppState) error
	GetDesiredState() (map[string]models.DesiredAppState, error)
	DeleteDesiredState(desiredStates ...models.DesiredAppState) error

	SaveActualState(actualStates ...models.InstanceHeartbeat) error

	SaveCrashCounts(crashCounts ...models.CrashCount) error

	SavePendingStartMessages(startMessages ...models.PendingStartMessage) error
	GetPendingStartMessages() (map[string]models.PendingStartMessage, error)
	DeletePendingStartMessages(startMessages ...models.PendingStartMessage) error

	SavePendingStopMessages(stopMessages ...models.PendingStopMessage) error
	GetPendingStopMessages() (map[string]models.PendingStopMessage, error)
	DeletePendingStopMessages(stopMessages ...models.PendingStopMessage) error
}

type RealStore struct {
	config  config.Config
	adapter storeadapter.StoreAdapter
	logger  logger.Logger
}

func NewStore(config config.Config, adapter storeadapter.StoreAdapter, logger logger.Logger) *RealStore {
	return &RealStore{
		config:  config,
		adapter: adapter,
		logger:  logger,
	}
}

func (store *RealStore) fetchNodesUnderDir(dir string) ([]storeadapter.StoreNode, error) {
	node, err := store.adapter.ListRecursively(dir)
	if err != nil {
		if err == storeadapter.ErrorKeyNotFound {
			return []storeadapter.StoreNode{}, nil
		}
		return []storeadapter.StoreNode{}, err
	}
	return node.ChildNodes, nil
}

// buckle up, here be dragons...

func (store *RealStore) save(stuff interface{}, root string, ttl uint64) error {
	t := time.Now()
	arrValue := reflect.ValueOf(stuff)

	nodes := make([]storeadapter.StoreNode, arrValue.Len())
	for i := 0; i < arrValue.Len(); i++ {
		item := arrValue.Index(i).Interface().(Storeable)
		nodes[i] = storeadapter.StoreNode{
			Key:   root + "/" + item.StoreKey(),
			Value: item.ToJSON(),
			TTL:   ttl,
		}
	}

	err := store.adapter.Set(nodes)

	store.logger.Info(fmt.Sprintf("Save Duration %s", root), map[string]string{
		"Number of Items": fmt.Sprintf("%d", arrValue.Len()),
		"Duration":        fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
	})
	return err
}

func (store *RealStore) get(root string, mapType reflect.Type, constructor reflect.Value) (reflect.Value, error) {
	t := time.Now()

	nodes, err := store.fetchNodesUnderDir(root)
	if err != nil {
		return reflect.MakeMap(mapType), err
	}

	mapToReturn := reflect.MakeMap(mapType)
	for _, node := range nodes {
		out := constructor.Call([]reflect.Value{reflect.ValueOf(node.Value)})
		if !out[1].IsNil() {
			return reflect.MakeMap(mapType), out[1].Interface().(error)
		}
		item := out[0].Interface().(Storeable)
		mapToReturn.SetMapIndex(reflect.ValueOf(item.StoreKey()), out[0])
	}

	store.logger.Info(fmt.Sprintf("Get Duration %s", root), map[string]string{
		"Number of Items": fmt.Sprintf("%d", mapToReturn.Len()),
		"Duration":        fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
	})
	return mapToReturn, nil
}

func (store *RealStore) delete(stuff interface{}, root string) error {
	t := time.Now()
	arrValue := reflect.ValueOf(stuff)

	for i := 0; i < arrValue.Len(); i++ {
		item := arrValue.Index(i).Interface().(Storeable)

		err := store.adapter.Delete(root + "/" + item.StoreKey())
		if err != nil {
			return err
		}
	}

	store.logger.Info(fmt.Sprintf("Delete Duration %s", root), map[string]string{
		"Number of Items": fmt.Sprintf("%d", arrValue.Len()),
		"Duration":        fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
	})

	return nil
}
