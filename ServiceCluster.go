package main

import (
	"errors"
	"sync"
	"github.com/golang/glog"
)

type ServiceCluster struct {
	instances []*Service
	lastIndex int
	lock      sync.RWMutex
}

func (cl *ServiceCluster) Next() (*Service, error) {
	if cl == nil {
		return nil, StatusError{}
	}
	cl.lock.RLock()
	defer cl.lock.RUnlock()
	if len(cl.instances) == 0 {
		return nil, errors.New("no alive instance found")
	}
	var instance *Service
	for tries := 0; tries < len(cl.instances); tries++ {
		index := (cl.lastIndex + 1) % len(cl.instances)
		cl.lastIndex = index

		instance = cl.instances[index]
		glog.V(5).Infof("Checking instance %d status : %s", index, instance.status.compute())
		if ( instance.status.compute() == STARTED_STATUS) {
			return instance, nil
		}
	}
	glog.V(5).Infof("No instance started for %s", instance.name)

	lastStatus := instance.status
	glog.V(5).Infof("Last status :")
	glog.V(5).Infof("   current  : %s", lastStatus.current)
	glog.V(5).Infof("   expected : %s", lastStatus.expected)
	glog.V(5).Infof("   alive : %s", lastStatus.alive)
	return nil, StatusError{instance.status.compute(), lastStatus }
}

func (cl *ServiceCluster) Remove(instanceIndex string) {

	match := -1
	for k, v := range cl.instances {
		if v.index == instanceIndex {
			match = k
		}
	}

	cl.instances = append(cl.instances[:match], cl.instances[match+1:]...)
	cl.Dump("remove")
}

// Get an service by its key (index). Returns nil if not found.
func (cl *ServiceCluster) Get(instanceIndex string) *Service {
	for i, v := range cl.instances {
		if v.index == instanceIndex {
			return cl.instances[i]
		}
	}
	return nil
}

func (cl *ServiceCluster) Add(service *Service) {
	for index, v := range cl.instances {
		if v.index == service.index {
			cl.instances[index] = service
			return
		}
	}

	cl.instances = append(cl.instances, service)
}

func (cl *ServiceCluster) Dump(action string) {
	for _, v := range cl.instances {
		glog.Infof("Dump after %s %s -> %s:%d", action, v.index, v.location.Host, v.location.Port)
	}
}
