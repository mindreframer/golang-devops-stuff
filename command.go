package main

import (
	"github.com/goraft/raft"
	"github.com/skynetservices/skydns/msg"
	"github.com/skynetservices/skydns/registry"
	"log"
	"time"
)

// Command for adding service to registry
type AddServiceCommand struct {
	Service msg.Service
}

// Creates a new AddServiceCommand
func NewAddServiceCommand(s msg.Service) *AddServiceCommand {
	s.Expires = getExpirationTime(s.TTL)

	return &AddServiceCommand{s}
}

// Name of command
func (c *AddServiceCommand) CommandName() string {
	return "add-service"
}

// Adds service to registry
func (c *AddServiceCommand) Apply(server raft.Server) (interface{}, error) {
	reg := server.Context().(registry.Registry)
	err := reg.Add(c.Service)

	if err == nil {
		log.Println("Added Service:", c.Service)
	}

	return c.Service, err
}

type UpdateTTLCommand struct {
	UUID    string
	TTL     uint32
	Expires time.Time
}

// NewUpdateTTLCommands returns a new UpdateTTLCommand
func NewUpdateTTLCommand(uuid string, ttl uint32) *UpdateTTLCommand {
	return &UpdateTTLCommand{uuid, ttl, getExpirationTime(ttl)}
}

// Name of command
func (c *UpdateTTLCommand) CommandName() string {
	return "update-ttl"
}

// Updates TTL in registry
func (c *UpdateTTLCommand) Apply(server raft.Server) (interface{}, error) {
	reg := server.Context().(registry.Registry)
	err := reg.UpdateTTL(c.UUID, c.TTL, c.Expires)

	if err == nil {
		log.Println("Updated Service TTL:", c.UUID, c.TTL)
	}

	return c.UUID, err
}

type RemoveServiceCommand struct {
	UUID string
}

// Creates a new RemoveServiceCommand
func NewRemoveServiceCommand(uuid string) *RemoveServiceCommand {
	return &RemoveServiceCommand{uuid}
}

// Name of command
func (c *RemoveServiceCommand) CommandName() string {
	return "remove-service"
}

// Updates TTL in registry
func (c *RemoveServiceCommand) Apply(server raft.Server) (interface{}, error) {

	reg := server.Context().(registry.Registry)
	err := reg.RemoveUUID(c.UUID)

	if err == nil {
		log.Println("Removed Service:", c.UUID)
	}

	return c.UUID, err
}

func getExpirationTime(ttl uint32) time.Time {
	return time.Now().Add(time.Duration(ttl) * time.Second)
}
