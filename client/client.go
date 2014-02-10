// Copyright (c) 2013 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"github.com/skynetservices/skydns/msg"
	"io"
	"net/http"
	"strconv"
)

var (
	ErrNoHttpAddress   = errors.New("No HTTP address specified")
	ErrNoDnsAddress    = errors.New("No DNS address specified")
	ErrInvalidResponse = errors.New("Invalid HTTP response")
	ErrServiceNotFound = errors.New("Service not found")
	ErrConflictingUUID = errors.New("Conflicting UUID")
)

type (
	Client struct {
		base    string
		secret  string
		h       *http.Client
		basedns string
		domain  string
		d       *dns.Client
		DNS     bool // if true use the DNS when listing servies
	}

	NameCount map[string]int
)

// NewClient creates a new skydns client with the specificed host address and
// DNS port.
func NewClient(base, secret, domain, basedns string) (*Client, error) {
	if base == "" {
		return nil, ErrNoHttpAddress
	}
	if basedns == "" {
		return nil, ErrNoDnsAddress
	}
	return &Client{
		base:    base,
		basedns: basedns,
		domain:  dns.Fqdn(domain),
		secret:  secret,
		h:       &http.Client{},
		d:       &dns.Client{},
	}, nil
}

func (c *Client) Add(uuid string, s *msg.Service) error {
	b := bytes.NewBuffer(nil)
	if err := json.NewEncoder(b).Encode(s); err != nil {
		return err
	}
	req, err := c.newRequest("PUT", c.joinUrl(uuid), b)
	if err != nil {
		return err
	}
	resp, err := c.h.Do(req)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusConflict:
		return ErrConflictingUUID
	default:
		return ErrInvalidResponse
	}
}

func (c *Client) Delete(uuid string) error {
	req, err := c.newRequest("DELETE", c.joinUrl(uuid), nil)
	if err != nil {
		return err
	}
	resp, err := c.h.Do(req)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	return nil
}

func (c *Client) Get(uuid string) (*msg.Service, error) {
	req, err := c.newRequest("GET", c.joinUrl(uuid), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.h.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	switch resp.StatusCode {
	case http.StatusOK:
		break
	case http.StatusNotFound:
		return nil, ErrServiceNotFound
	default:
		return nil, ErrInvalidResponse
	}

	var s *msg.Service
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, err
	}
	return s, nil
}

func (c *Client) Update(uuid string, ttl uint32) error {
	b := bytes.NewBuffer([]byte(fmt.Sprintf(`{"TTL":%d}`, ttl)))
	req, err := c.newRequest("PATCH", c.joinUrl(uuid), b)
	if err != nil {
		return err
	}
	resp, err := c.h.Do(req)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	return nil
}

func (c *Client) GetAllServices() ([]*msg.Service, error) {
	req, err := c.newRequest("GET", c.joinUrl(""), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.h.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	var out []*msg.Service
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (c *Client) GetAllServicesDNS() ([]*msg.Service, error) {
	req, err := c.newRequestDNS("", dns.TypeSRV)
	if err != nil {
		return nil, err
	}
	resp, _, err := c.d.Exchange(req, c.basedns)
	if err != nil {
		return nil, err
	}
	// Handle UUID.skydns.local additional section stuff? TODO(miek)
	s := make([]*msg.Service, len(resp.Answer))
	for i, r := range resp.Answer {
		if v, ok := r.(*dns.SRV); ok {
			s[i] = &msg.Service{
				// TODO(miek): uehh, stuff it in Name?
				Name: v.Header().Name + " (Priority: " + strconv.Itoa(int(v.Priority)) + ", " + "Weight: " + strconv.Itoa(int(v.Weight)) + ")",
				Host: v.Target,
				Port: v.Port,
				TTL:  r.Header().Ttl,
			}
		}
	}
	return s, nil
}

func (c *Client) GetRegions() (NameCount, error) {
	req, err := c.newRequest("GET", fmt.Sprintf("%s/skydns/regions/", c.base), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.h.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	var out NameCount
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetEnvironments() (NameCount, error) {
	req, err := c.newRequest("GET", fmt.Sprintf("%s/skydns/environments/", c.base), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.h.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	var out NameCount
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) AddCallback(uuid string, cb *msg.Callback) error {
	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(cb); err != nil {
		return err
	}
	req, err := c.newRequest("PUT", fmt.Sprintf("%s/skydns/callbacks/%s", c.base, uuid), buf)
	if err != nil {
		return err
	}
	resp, err := c.h.Do(req)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusNotFound:
		return ErrServiceNotFound
	default:
		return ErrInvalidResponse
	}
}

func (c *Client) joinUrl(uuid string) string {
	return fmt.Sprintf("%s/skydns/services/%s", c.base, uuid)
}

func (c *Client) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if c.secret != "" {
		req.Header.Add("Authorization", c.secret)
	}
	return req, err
}

func (c *Client) newRequestDNS(qname string, qtype uint16) (*dns.Msg, error) {
	m := new(dns.Msg)
	if qname == "" {
		m.SetQuestion(c.domain, qtype)
	} else {
		m.SetQuestion(qname+"."+c.domain, qtype)
	}
	return m, nil
}
