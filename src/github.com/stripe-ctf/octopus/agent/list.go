package agent

import (
	"github.com/stripe-ctf/octopus/state"
)

// List of agents that can be manipulated as a group
type List []*Agent

func NewList() List {
	list := make([]*Agent, state.NodeCount())
	for i := 0; i < state.NodeCount(); i++ {
		list[i] = NewAgent(uint(i))
	}
	return list
}

func (l List) Start() {
	for _, agent := range l {
		agent.Start()
	}
}

func (l List) Dryrun() {
	for _, agent := range l {
		agent.Dryrun()
	}
}

func (l List) Prepare() {
	for _, agent := range l {
		agent.Prepare()
	}
}
