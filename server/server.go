// Copyright (c) 2013 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/goraft/raft"
	"github.com/gorilla/mux"
	"github.com/miekg/dns"
	"github.com/skynetservices/skydns/msg"
	"github.com/skynetservices/skydns/registry"
	"github.com/skynetservices/skydns/stats"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

/* TODO:
   Set Priority based on Region
   Dynamically set Weight/Priority in DNS responses
   Handle API call for setting host statistics
   Handle Errors in DNS
   Master should cleanup expired services
   TTL cleanup thread should shutdown/start based on being elected master
*/

func init() {
	// Register Raft Commands
	raft.RegisterCommand(&AddServiceCommand{})
	raft.RegisterCommand(&UpdateTTLCommand{})
	raft.RegisterCommand(&RemoveServiceCommand{})
	raft.RegisterCommand(&AddCallbackCommand{})
}

type Server struct {
	members      []string // initial members to join with
	nameservers  []string // nameservers to forward to
	domain       string
	dnsAddr      string
	httpAddr     string
	readTimeout  time.Duration
	writeTimeout time.Duration
	waiter       *sync.WaitGroup

	registry registry.Registry

	dnsUDPServer *dns.Server
	dnsTCPServer *dns.Server
	dnsHandler   *dns.ServeMux

	httpServer *http.Server
	router     *mux.Router

	raftServer raft.Server
	dataDir    string
	secret     string
}

// Newserver returns a new Server.
func NewServer(members []string, domain string, dnsAddr string, httpAddr string, dataDir string, rt, wt time.Duration, secret string, nameservers []string) (s *Server) {
	s = &Server{
		members:      members,
		domain:       domain,
		dnsAddr:      dnsAddr,
		httpAddr:     httpAddr,
		readTimeout:  rt,
		writeTimeout: wt,
		router:       mux.NewRouter(),
		registry:     registry.New(),
		dataDir:      dataDir,
		dnsHandler:   dns.NewServeMux(),
		waiter:       new(sync.WaitGroup),
		secret:       secret,
		nameservers:  nameservers,
	}

	if _, err := os.Stat(s.dataDir); os.IsNotExist(err) {
		log.Fatal("Data directory does not exist: ", dataDir)
		return
	}

	// DNS
	s.dnsHandler.Handle(".", s)

	authWrapper := s.authHTTPWrapper

	// API Routes
	s.router.HandleFunc("/skydns/services/{uuid}", authWrapper(s.addServiceHTTPHandler)).Methods("PUT")
	s.router.HandleFunc("/skydns/services/{uuid}", authWrapper(s.getServiceHTTPHandler)).Methods("GET")
	s.router.HandleFunc("/skydns/services/{uuid}", authWrapper(s.removeServiceHTTPHandler)).Methods("DELETE")
	s.router.HandleFunc("/skydns/services/{uuid}", authWrapper(s.updateServiceHTTPHandler)).Methods("PATCH")

	s.router.HandleFunc("/skydns/callbacks/{uuid}", authWrapper(s.addCallbackHTTPHandler)).Methods("PUT")

	// External API Routes
	// /skydns/services #list all services
	s.router.HandleFunc("/skydns/services/", authWrapper(s.getServicesHTTPHandler)).Methods("GET")
	// /skydns/regions #list all regions
	s.router.HandleFunc("/skydns/regions/", authWrapper(s.getRegionsHTTPHandler)).Methods("GET")
	// /skydns/environnments #list all environments
	s.router.HandleFunc("/skydns/environments/", authWrapper(s.getEnvironmentsHTTPHandler)).Methods("GET")

	// Raft Routes
	s.router.HandleFunc("/raft/join", s.joinHandler).Methods("POST")

	return
}

// DNSAddr returns IP:Port of a DNS Server.
func (s *Server) DNSAddr() string { return s.dnsAddr }

// HTTPAddr returns IP:Port of HTTP Server.
func (s *Server) HTTPAddr() string { return s.httpAddr }

// Start starts a DNS server and blocks waiting to be killed.
func (s *Server) Start() (*sync.WaitGroup, error) {
	var err error
	log.Printf("Initializing Server. DNS Addr: %q, HTTP Addr: %q, Data Dir: %q, Forwarders: %q", s.dnsAddr, s.httpAddr, s.dataDir, s.nameservers)

	// Initialize and start Raft server.
	transporter := raft.NewHTTPTransporter("/raft")
	s.raftServer, err = raft.NewServer(s.HTTPAddr(), s.dataDir, transporter, nil, s.registry, "")
	if err != nil {
		log.Fatal(err)
	}
	transporter.Install(s.raftServer, s)
	s.raftServer.Start()

	// Join to leader if specified.
	if len(s.members) > 0 {
		log.Println("Joining cluster:", strings.Join(s.members, ","))

		if !s.raftServer.IsLogEmpty() {
			log.Fatal("Cannot join with an existing log")
		}

		if err := s.Join(s.members); err != nil {
			return nil, err
		}

		log.Println("Joined cluster")

		// Initialize the server by joining itself.
	} else if s.raftServer.IsLogEmpty() {
		log.Println("Initializing new cluster")

		_, err := s.raftServer.Do(&raft.DefaultJoinCommand{
			Name:             s.raftServer.Name(),
			ConnectionString: s.connectionString(),
		})

		if err != nil {
			log.Fatal(err)
			return nil, err
		}

	} else {
		log.Println("Recovered from log")
	}

	s.dnsTCPServer = &dns.Server{
		Addr:         s.DNSAddr(),
		Net:          "tcp",
		Handler:      s.dnsHandler,
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
	}

	s.dnsUDPServer = &dns.Server{
		Addr:         s.DNSAddr(),
		Net:          "udp",
		Handler:      s.dnsHandler,
		UDPSize:      65535,
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
	}

	s.httpServer = &http.Server{
		Addr:           s.HTTPAddr(),
		Handler:        s.router,
		ReadTimeout:    s.readTimeout,
		WriteTimeout:   s.writeTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	go s.listenAndServe()

	s.waiter.Add(1)
	go s.run()

	return s.waiter, nil
}

// Stop stops a server.
func (s *Server) Stop() {
	log.Println("Stopping server")
	s.waiter.Done()
}

// Leader returns the current leader.
func (s *Server) Leader() string {
	l := s.raftServer.Leader()
	if l == "" {
		// We are a single node cluster, we are the leader
		return s.raftServer.Name()
	}
	return l
}

// IsLeader returns true if this instance the current leader.
func (s *Server) IsLeader() bool {
	return s.raftServer.State() == raft.Leader
}

// Members returns the current members.
func (s *Server) Members() (members []string) {
	peers := s.raftServer.Peers()

	for _, p := range peers {
		members = append(members, strings.TrimPrefix(p.ConnectionString, "http://"))
	}

	return
}

func (s *Server) run() {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	tick := time.Tick(1 * time.Second)

run:
	for {
		select {
		case <-tick:
			// We are the leader, we are responsible for managing TTLs
			if s.IsLeader() {
				expired := s.registry.GetExpired()

				// TODO: Possible race condition? We could be demoted while iterating
				// probably minimal chance of this happening, this will just cause commands to fail,
				// and new leader will take over anyway
				for _, uuid := range expired {
					stats.ExpiredCount.Inc(1)
					s.raftServer.Do(NewRemoveServiceCommand(uuid))
				}
			}
		case <-sig:
			break run
		}
	}
	s.Stop()
}

// Join joins an existing SkyDNS cluster.
func (s *Server) Join(members []string) error {
	command := &raft.DefaultJoinCommand{
		Name:             s.raftServer.Name(),
		ConnectionString: s.connectionString(),
	}

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(command)

	for _, m := range members {
		log.Println("Attempting to connect to:", m)

		resp, err := http.Post(fmt.Sprintf("http://%s/raft/join", strings.TrimSpace(m)), "application/json", &b)
		log.Println("Post returned")

		if err != nil {
			if _, ok := err.(*url.Error); ok {
				// If we receive a network error try the next member
				continue
			}

			return err
		}

		resp.Body.Close()
		return nil
	}

	return errors.New("Could not connect to any cluster members")
}

// HandleFunc proxies HTTP handlers to Gorilla's mux.Router.
func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.router.HandleFunc(pattern, handler)
}

// Handles incoming RAFT joins.
func (s *Server) joinHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("Processing incoming join")
	command := &raft.DefaultJoinCommand{}

	if err := json.NewDecoder(req.Body).Decode(&command); err != nil {
		log.Println("Error decoding json message:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := s.raftServer.Do(command); err != nil {
		switch err {
		case raft.NotLeaderError:
			log.Println("Redirecting to leader")
			s.redirectToLeader(w, req)
		default:
			log.Println("Error processing join:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// ServeDNS is the handler for DNS requests, responsible for parsing DNS request, possibly forwarding
// it to a real dns server and returning a response.
func (s *Server) ServeDNS(w dns.ResponseWriter, req *dns.Msg) {
	stats.RequestCount.Inc(1)

	q := req.Question[0]
	log.Printf("Received DNS Request for %q from %q", q.Name, w.RemoteAddr())

	// If the query does not fall in our s.domain, forward it
	if !strings.HasSuffix(q.Name, dns.Fqdn(s.domain)) {
		s.ServeDNSForward(w, req)
		return
	}
	m := new(dns.Msg)
	m.SetReply(req)
	m.Authoritative = true
	m.RecursionAvailable = true
	m.Answer = make([]dns.RR, 0, 10)
	defer w.WriteMsg(m)

	if q.Qtype == dns.TypeANY || q.Qtype == dns.TypeSRV {
		records, extra, err := s.getSRVRecords(q)

		if err != nil {
			// We are authoritative for this name, but it does not exist: NXDOMAIN
			m.SetRcode(req, dns.RcodeNameError)
			m.Ns = s.createSOA()
			log.Println("Error: ", err)
			return
		}

		m.Answer = append(m.Answer, records...)
		m.Extra = append(m.Extra, extra...)
	}

	if q.Qtype == dns.TypeA || q.Qtype == dns.TypeAAAA {
		records, err := s.getARecords(q)

		if err != nil {
			m.SetRcode(req, dns.RcodeNameError)
			m.Ns = s.createSOA()
			log.Println("Error: ", err)
			return
		}
		m.Answer = append(m.Answer, records...)
	}
	if len(m.Answer) == 0 { // Send back a NODATA response
		m.Ns = s.createSOA()
	}
}

// ServeDNSForward forwards a request to a nameservers and returns the response.
func (s *Server) ServeDNSForward(w dns.ResponseWriter, req *dns.Msg) {
	if len(s.nameservers) == 0 {
		log.Printf("Error: Failure to Forward DNS Request, no servers configured %q", dns.ErrServ)
		m := new(dns.Msg)
		m.SetReply(req)
		m.SetRcode(req, dns.RcodeServerFailure)
		m.Authoritative = false     // no matter what set to false
		m.RecursionAvailable = true // and this is still true
		w.WriteMsg(m)
		return
	}
	network := "udp"
	if _, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		network = "tcp"
	}
	c := &dns.Client{Net: network, ReadTimeout: 5 * time.Second}

	// Use request Id for "random" nameserver selection
	nsid := int(req.Id) % len(s.nameservers)
	try := 0
Redo:
	r, _, err := c.Exchange(req, s.nameservers[nsid])
	if err == nil {
		log.Printf("Forwarded DNS Request %q to %q", req.Question[0].Name, s.nameservers[nsid])
		w.WriteMsg(r)
		return
	}
	// Seen an error, this can only mean, "server not reached", try again
	// but only if we have not exausted our nameservers
	if try < len(s.nameservers) {
		log.Printf("Error: Failure to Forward DNS Request %q to %q", err, s.nameservers[nsid])
		try++
		nsid = (nsid + 1) % len(s.nameservers)
		goto Redo
	}

	log.Printf("Error: Failure to Forward DNS Request %q", err)
	m := new(dns.Msg)
	m.SetReply(req)
	m.SetRcode(req, dns.RcodeServerFailure)
	w.WriteMsg(m)
}

func (s *Server) getARecords(q dns.Question) (records []dns.RR, err error) {
	var h string
	name := strings.TrimSuffix(q.Name, ".")

	if name == s.domain {
		for _, m := range s.Members() {
			h, _, err = net.SplitHostPort(m)

			if err != nil {
				return
			}
			if q.Qtype == dns.TypeA {
				records = append(records, &dns.A{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 15}, A: net.ParseIP(h)})
			}
		}
	}
	// Leader should always be listed
	if name == "leader."+s.domain || name == "master."+s.domain || name == s.domain {
		h, _, err = net.SplitHostPort(s.Leader())
		if err != nil {
			return
		}
		if q.Qtype == dns.TypeA {
			records = append(records, &dns.A{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 15}, A: net.ParseIP(h)})
		}
		return
	}

	var (
		services []msg.Service
		key      = strings.TrimSuffix(q.Name, s.domain+".")
	)

	services, err = s.registry.Get(key)
	if err != nil {
		return
	}

	for _, serv := range services {
		ip := net.ParseIP(serv.Host)
		switch {
		case ip == nil:
			continue
		case ip.To4() != nil && q.Qtype == dns.TypeA:
			records = append(records, &dns.A{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: serv.TTL}, A: ip.To4()})
		case ip.To4() == nil && q.Qtype == dns.TypeAAAA:
			records = append(records, &dns.AAAA{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: serv.TTL}, AAAA: ip.To16()})
		}
	}
	return
}

func (s *Server) getSRVRecords(q dns.Question) (records []dns.RR, extra []dns.RR, err error) {
	var weight uint16
	services := make([]msg.Service, 0)

	key := strings.TrimSuffix(q.Name, s.domain+".")
	services, err = s.registry.Get(key)

	if err != nil {
		return
	}

	weight = 0
	if len(services) > 0 {
		weight = uint16(math.Floor(float64(100 / len(services))))
	}

	for _, serv := range services {
		// TODO: Dynamically set weight
		// a Service may have an IP as its Host"name", in this case
		// substitute UUID + "." + s.domain+"." an add an A record
		// with the name and IP in the additional section.
		// TODO(miek): check if resolvers actually grok this
		ip := net.ParseIP(serv.Host)
		switch {
		case ip == nil:
			records = append(records, &dns.SRV{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: serv.TTL},
				Priority: 10, Weight: weight, Port: serv.Port, Target: serv.Host + "."})
			continue
		case ip.To4() != nil:
			extra = append(extra, &dns.A{Hdr: dns.RR_Header{Name: serv.UUID + "." + s.domain + ".", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: serv.TTL}, A: ip.To4()})
			records = append(records, &dns.SRV{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: serv.TTL},
				Priority: 10, Weight: weight, Port: serv.Port, Target: serv.UUID + "." + s.domain + "."})
		case ip.To16() != nil:
			extra = append(extra, &dns.AAAA{Hdr: dns.RR_Header{Name: serv.UUID + "." + s.domain + ".", Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: serv.TTL}, AAAA: ip.To16()})
			records = append(records, &dns.SRV{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: serv.TTL},
				Priority: 10, Weight: weight, Port: serv.Port, Target: serv.UUID + "." + s.domain + "."})
		default:
			panic("skydns: internal error")
		}
	}

	// Append matching entries in different region than requested with a higher priority
	labels := dns.SplitDomainName(key)

	pos := len(labels) - 4
	if len(labels) >= 4 && labels[pos] != "*" {
		region := labels[pos]
		labels[pos] = "*"

		// TODO: This is pretty much a copy of the above, and should be abstracted
		additionalServices := make([]msg.Service, len(services))
		additionalServices, err = s.registry.Get(strings.Join(labels, "."))

		if err != nil {
			return
		}

		weight = 0
		if len(additionalServices) <= len(services) {
			return
		}

		weight = uint16(math.Floor(float64(100 / (len(additionalServices) - len(services)))))
		for _, serv := range additionalServices {
			// Exclude entries we already have
			if strings.ToLower(serv.Region) == region {
				continue
			}
			// TODO: Dynamically set priority and weight
			// TODO(miek): same as above: abstract away
			ip := net.ParseIP(serv.Host)
			switch {
			case ip == nil:
				records = append(records, &dns.SRV{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: serv.TTL},
					Priority: 20, Weight: weight, Port: serv.Port, Target: serv.Host + "."})
				continue
			case ip.To4() != nil:
				extra = append(extra, &dns.A{Hdr: dns.RR_Header{Name: serv.UUID + "." + s.domain + ".", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: serv.TTL}, A: ip.To4()})
				records = append(records, &dns.SRV{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: serv.TTL},
					Priority: 20, Weight: weight, Port: serv.Port, Target: serv.UUID + "." + s.domain + "."})
			case ip.To16() != nil:
				extra = append(extra, &dns.AAAA{Hdr: dns.RR_Header{Name: serv.UUID + "." + s.domain + ".", Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: serv.TTL}, AAAA: ip.To16()})
				records = append(records, &dns.SRV{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: serv.TTL},
					Priority: 20, Weight: weight, Port: serv.Port, Target: serv.UUID + "." + s.domain + "."})
			default:
				panic("skydns: internal error")
			}
		}
	}
	return
}

// Returns the connection string.
func (s *Server) connectionString() string {
	return fmt.Sprintf("http://%s", s.httpAddr)
}

// Binds to DNS and HTTP ports and starts accepting connections
func (s *Server) listenAndServe() {
	go func() {
		err := s.dnsTCPServer.ListenAndServe()
		if err != nil {
			log.Fatalf("Start %s listener on %s failed:%s", s.dnsTCPServer.Net, s.dnsTCPServer.Addr, err.Error())
		}
	}()

	go func() {
		err := s.dnsUDPServer.ListenAndServe()
		if err != nil {
			log.Fatalf("Start %s listener on %s failed:%s", s.dnsUDPServer.Net, s.dnsUDPServer.Addr, err.Error())
		}
	}()

	go func() {
		err := s.httpServer.ListenAndServe()
		if err != nil {
			log.Fatalf("Start http listener on %s failed:%s", s.httpServer.Addr, err.Error())
		}
	}()
}

func (s *Server) redirectToLeader(w http.ResponseWriter, req *http.Request) {
	if s.Leader() != "" {
		http.Redirect(w, req, "http://"+s.Leader()+req.URL.Path, http.StatusMovedPermanently)
	} else {
		log.Println("Error: Leader Unknown")
		http.Error(w, "Leader unknown", http.StatusInternalServerError)
	}
}

// shared auth method on server.
func (s *Server) authenticate(secret string) (err error) {
	if s.secret != "" && secret != s.secret {
		err = errors.New("Forbidden")
	}
	return
}

// Handle API add service requests
func (s *Server) addServiceHTTPHandler(w http.ResponseWriter, req *http.Request) {
	stats.AddServiceCount.Inc(1)
	vars := mux.Vars(req)

	var uuid string
	var ok bool

	if uuid, ok = vars["uuid"]; !ok {
		http.Error(w, "UUID required", http.StatusBadRequest)
		return
	}

	var serv msg.Service

	if err := json.NewDecoder(req.Body).Decode(&serv); err != nil {
		log.Println("Error: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if serv.Host == "" || serv.Port == 0 {
		http.Error(w, "Host and Port required", http.StatusBadRequest)
		return
	}

	serv.UUID = uuid

	if _, err := s.raftServer.Do(NewAddServiceCommand(serv)); err != nil {
		switch err {
		case registry.ErrExists:
			http.Error(w, err.Error(), http.StatusConflict)
		case raft.NotLeaderError:
			s.redirectToLeader(w, req)
		default:
			log.Println("Error: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Handle API remove service requests
func (s *Server) removeServiceHTTPHandler(w http.ResponseWriter, req *http.Request) {
	stats.RemoveServiceCount.Inc(1)
	vars := mux.Vars(req)

	var uuid string
	var ok bool

	if uuid, ok = vars["uuid"]; !ok {
		http.Error(w, "UUID required", http.StatusBadRequest)
		return
	}

	if _, err := s.raftServer.Do(NewRemoveServiceCommand(uuid)); err != nil {

		switch err {
		case registry.ErrNotExists:
			http.Error(w, err.Error(), http.StatusNotFound)
		case raft.NotLeaderError:
			s.redirectToLeader(w, req)
		default:
			log.Println("Error: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// Handle API update service requests
func (s *Server) updateServiceHTTPHandler(w http.ResponseWriter, req *http.Request) {
	stats.UpdateTTLCount.Inc(1)
	vars := mux.Vars(req)

	var uuid string
	var ok bool

	if uuid, ok = vars["uuid"]; !ok {
		http.Error(w, "UUID required", http.StatusBadRequest)
		return
	}

	var serv msg.Service
	if err := json.NewDecoder(req.Body).Decode(&serv); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := s.raftServer.Do(NewUpdateTTLCommand(uuid, serv.TTL)); err != nil {
		switch err {
		case registry.ErrNotExists:
			http.Error(w, err.Error(), http.StatusNotFound)
		case raft.NotLeaderError:
			s.redirectToLeader(w, req)
		default:
			log.Println("Error: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// Handle API get service requests
func (s *Server) getServiceHTTPHandler(w http.ResponseWriter, req *http.Request) {
	stats.GetServiceCount.Inc(1)
	vars := mux.Vars(req)

	var uuid string
	var ok bool

	if uuid, ok = vars["uuid"]; !ok {
		http.Error(w, "UUID required", http.StatusBadRequest)
		return
	}

	log.Println("Retrieving Service ", uuid)
	serv, err := s.registry.GetUUID(uuid)

	if err != nil {
		switch err {
		case registry.ErrNotExists:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			log.Println("Error: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	if err := json.NewEncoder(w).Encode(serv); err != nil {
		log.Println("Error: ", err)
	}
}

// secrethttphandlerwrapper will wrap a standard handler
// if the secret is specified for the server
func (s *Server) authHTTPWrapper(handler http.HandlerFunc) http.HandlerFunc {
	if s.secret != "" {
		return func(w http.ResponseWriter, req *http.Request) {
			//read the authorization header to get the secret.
			secret := req.Header.Get("Authorization")

			if err := s.authenticate(secret); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
			handler(w, req)
		}
	}
	return handler
}

// Return a SOA record for this SkyDNS instance
func (s *Server) createSOA() []dns.RR {
	dom := dns.Fqdn(s.domain)
	soa := &dns.SOA{Hdr: dns.RR_Header{Name: dom, Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: 3600},
		Ns:      "master." + dom,
		Mbox:    "hostmaster." + dom,
		Serial:  uint32(time.Now().Unix()),
		Refresh: 28800,
		Retry:   7200,
		Expire:  604800,
		Minttl:  3600,
	}
	return []dns.RR{soa}
}
