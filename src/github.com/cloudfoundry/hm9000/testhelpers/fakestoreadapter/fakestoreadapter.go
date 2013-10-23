package fakestoreadapter

import (
	"github.com/cloudfoundry/hm9000/storeadapter"
	"regexp"
	"strings"
)

type containerNode struct {
	dir   bool
	nodes map[string]*containerNode

	storeNode storeadapter.StoreNode
}

type FakeStoreAdapterErrorInjector struct {
	KeyRegexp *regexp.Regexp
	Error     error
}

func NewFakeStoreAdapterErrorInjector(keyRegexp string, err error) *FakeStoreAdapterErrorInjector {
	return &FakeStoreAdapterErrorInjector{
		KeyRegexp: regexp.MustCompile(keyRegexp),
		Error:     err,
	}
}

type FakeStoreAdapter struct {
	DidConnect    bool
	DidDisconnect bool

	ConnectErr        error
	DisconnectErr     error
	SetErrInjector    *FakeStoreAdapterErrorInjector
	GetErrInjector    *FakeStoreAdapterErrorInjector
	ListErrInjector   *FakeStoreAdapterErrorInjector
	DeleteErrInjector *FakeStoreAdapterErrorInjector

	rootNode *containerNode
}

func New() *FakeStoreAdapter {
	adapter := &FakeStoreAdapter{}
	adapter.Reset()
	return adapter
}

func (adapter *FakeStoreAdapter) Reset() {
	adapter.DidConnect = false
	adapter.DidDisconnect = false

	adapter.ConnectErr = nil
	adapter.DisconnectErr = nil
	adapter.SetErrInjector = nil
	adapter.GetErrInjector = nil
	adapter.ListErrInjector = nil
	adapter.DeleteErrInjector = nil

	adapter.rootNode = &containerNode{
		dir:   true,
		nodes: make(map[string]*containerNode),
	}
}

func (adapter *FakeStoreAdapter) Connect() error {
	adapter.DidConnect = true
	return adapter.ConnectErr
}

func (adapter *FakeStoreAdapter) Disconnect() error {
	adapter.DidDisconnect = true
	return adapter.DisconnectErr
}

func (adapter *FakeStoreAdapter) Set(nodes []storeadapter.StoreNode) error {
	for _, node := range nodes {
		if adapter.SetErrInjector != nil && adapter.SetErrInjector.KeyRegexp.MatchString(node.Key) {
			return adapter.SetErrInjector.Error
		}
		components := adapter.keyComponents(node.Key)

		container := adapter.rootNode
		for i, component := range components {
			if i == len(components)-1 {
				existingNode, exists := container.nodes[component]
				if exists && existingNode.dir {
					return storeadapter.ErrorNodeIsDirectory
				}
				container.nodes[component] = &containerNode{storeNode: node}
			} else {
				existingNode, exists := container.nodes[component]
				if exists {
					if !existingNode.dir {
						return storeadapter.ErrorNodeIsNotDirectory
					}
					container = existingNode
				} else {
					newContainer := &containerNode{dir: true, nodes: make(map[string]*containerNode)}
					container.nodes[component] = newContainer
					container = newContainer
				}
			}
		}
	}
	return nil
}

func (adapter *FakeStoreAdapter) Get(key string) (storeadapter.StoreNode, error) {
	if adapter.GetErrInjector != nil && adapter.GetErrInjector.KeyRegexp.MatchString(key) {
		return storeadapter.StoreNode{}, adapter.GetErrInjector.Error
	}

	components := adapter.keyComponents(key)
	container := adapter.rootNode
	for _, component := range components {
		var exists bool
		container, exists = container.nodes[component]
		if !exists {
			return storeadapter.StoreNode{}, storeadapter.ErrorKeyNotFound
		}
	}

	if container.dir {
		return storeadapter.StoreNode{}, storeadapter.ErrorNodeIsDirectory
	} else {
		return container.storeNode, nil
	}
}

func (adapter *FakeStoreAdapter) ListRecursively(key string) (storeadapter.StoreNode, error) {
	if adapter.ListErrInjector != nil && adapter.ListErrInjector.KeyRegexp.MatchString(key) {
		return storeadapter.StoreNode{}, adapter.ListErrInjector.Error
	}

	container := adapter.rootNode

	components := adapter.keyComponents(key)
	for _, component := range components {
		var exists bool
		container, exists = container.nodes[component]
		if !exists {
			return storeadapter.StoreNode{}, storeadapter.ErrorKeyNotFound
		}
	}

	if !container.dir {
		return storeadapter.StoreNode{}, storeadapter.ErrorNodeIsNotDirectory
	}

	return adapter.listContainerNode(key, container), nil
}

func (adapter *FakeStoreAdapter) listContainerNode(key string, container *containerNode) storeadapter.StoreNode {
	childNodes := []storeadapter.StoreNode{}

	for nodeKey, node := range container.nodes {
		if node.dir {
			if key == "/" {
				nodeKey = "/" + nodeKey
			} else {
				nodeKey = key + "/" + nodeKey
			}
			childNodes = append(childNodes, adapter.listContainerNode(nodeKey, node))
		} else {
			childNodes = append(childNodes, node.storeNode)
		}
	}

	return storeadapter.StoreNode{
		Key:        key,
		Dir:        true,
		ChildNodes: childNodes,
	}
}

func (adapter *FakeStoreAdapter) Delete(key string) error {
	if adapter.DeleteErrInjector != nil && adapter.DeleteErrInjector.KeyRegexp.MatchString(key) {
		return adapter.DeleteErrInjector.Error
	}

	components := adapter.keyComponents(key)
	container := adapter.rootNode
	parentNode := adapter.rootNode
	for _, component := range components {
		var exists bool
		parentNode = container
		container, exists = container.nodes[component]
		if !exists {
			return storeadapter.ErrorKeyNotFound
		}
	}

	if container.dir {
		return storeadapter.ErrorNodeIsDirectory
	} else {
		delete(parentNode.nodes, components[len(components)-1])
		return nil
	}
}

func (adapter *FakeStoreAdapter) keyComponents(key string) (components []string) {
	for _, s := range strings.Split(key, "/") {
		if s != "" {
			components = append(components, s)
		}
	}

	return components
}
