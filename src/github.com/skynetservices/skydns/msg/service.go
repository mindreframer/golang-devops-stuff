// Copyright (c) 2013 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package msg

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Service struct {
	UUID        string
	Name        string
	Version     string
	Environment string
	Region      string
	Host        string
	Port        uint16
	TTL         uint32 // Seconds
	Expires     time.Time
	Callback    map[string]Callback `json:"-"` // Callbacks are found by UUID
}

// RemainingTTL returns the amount of time remaining before expiration.
func (s *Service) RemainingTTL() uint32 {
	d := s.Expires.Sub(time.Now())
	ttl := uint32(d.Seconds())

	if ttl < 1 {
		return 0
	}
	return ttl
}

// UpdateTTL updates the TTL property to the RemainingTTL.
func (s *Service) UpdateTTL() {
	s.TTL = s.RemainingTTL()
}

type Callback struct {
	UUID string

	// Name of the service
	Name        string
	Version     string
	Environment string
	Region      string
	Host        string

	Reply string
	Port  uint16
}

// Call calls the callback and performs the HTTP request.
func (c Callback) Call(s Service) {
	b, err := json.Marshal(s)
	if err != nil {
		return
	}
	req, err := http.NewRequest("DELETE", "http://"+c.Reply+":"+strconv.Itoa(int(c.Port))+"/skydns/callbacks/"+c.UUID, bytes.NewBuffer(b))
	if err != nil {
		log.Println("Failed to create req.", err.Error)
		return
	}
	if resp, err := http.DefaultClient.Do(req); err == nil {
		resp.Body.Close()
	}
	log.Println("Performed callback to:", c.Reply, c.Port)
	return
}
