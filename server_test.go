package main

import (
	"bytes"
	"encoding/json"
	"github.com/miekg/dns"
	"github.com/skynetservices/skydns/msg"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"
)

/* TODO: Tests
   Test Cluster
   Test TTL expiration
   Benchmarks
*/

func TestAddService(t *testing.T) {
	s := newTestServer("", 9500, 9501, "")
	defer s.Stop()

	m := msg.Service{
		Name:        "TestService",
		Version:     "1.0.0",
		Region:      "Test",
		Host:        "localhost",
		Environment: "Production",
		Port:        9000,
		TTL:         4,
	}

	b, err := json.Marshal(m)

	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("PUT", "/skydns/services/123", bytes.NewBuffer(b))
	resp := httptest.NewRecorder()

	s.router.ServeHTTP(resp, req)

	if resp.Code != 201 || s.registry.Len() != 1 {
		t.Fatal("Failed to add service")
	}
}

func TestAddServiceDuplicate(t *testing.T) {
	s := newTestServer("", 9510, 9511, "")
	defer s.Stop()

	m := msg.Service{
		Name:        "TestService",
		Version:     "1.0.0",
		Region:      "Test",
		Host:        "localhost",
		Environment: "Production",
		Port:        9000,
		TTL:         4,
	}

	b, err := json.Marshal(m)

	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("PUT", "/skydns/services/123", bytes.NewBuffer(b))
	resp := httptest.NewRecorder()

	s.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated || s.registry.Len() != 1 {
		t.Fatal("Failed to add service")
	}

	req, _ = http.NewRequest("PUT", "/skydns/services/123", bytes.NewBuffer(b))
	resp = httptest.NewRecorder()
	s.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict || s.registry.Len() != 1 {
		t.Fatal("Duplicates should return error code 407", resp.Code)
	}
}

func TestRemoveService(t *testing.T) {
	s := newTestServer("", 9520, 9521, "")
	defer s.Stop()

	m := msg.Service{
		UUID:        "123",
		Name:        "TestService",
		Version:     "1.0.0",
		Region:      "Test",
		Host:        "localhost",
		Environment: "Production",
		Port:        9000,
		TTL:         4,
	}

	s.registry.Add(m)

	req, _ := http.NewRequest("DELETE", "/skydns/services/"+m.UUID, nil)
	resp := httptest.NewRecorder()

	s.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK || s.registry.Len() != 0 {
		t.Fatal("Failed to remove service")
	}
}

func TestRemoveUnknownService(t *testing.T) {
	s := newTestServer("", 9530, 9531, "")
	defer s.Stop()

	m := msg.Service{
		UUID:        "123",
		Name:        "TestService",
		Version:     "1.0.0",
		Region:      "Test",
		Host:        "localhost",
		Environment: "Production",
		Port:        9000,
		TTL:         4,
	}

	s.registry.Add(m)

	req, _ := http.NewRequest("DELETE", "/skydns/services/54321", nil)
	resp := httptest.NewRecorder()

	s.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound || s.registry.Len() != 1 {
		t.Fatal("API should return 404 when removing unknown service")
	}
}

func TestUpdateTTL(t *testing.T) {
	s := newTestServer("", 9540, 9541, "")
	defer s.Stop()

	m := msg.Service{
		UUID:        "123",
		Name:        "TestService",
		Version:     "1.0.0",
		Region:      "Test",
		Host:        "localhost",
		Environment: "Production",
		Port:        9000,
		TTL:         4,
	}

	s.registry.Add(m)

	m.TTL = 25
	b, err := json.Marshal(m)

	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("PATCH", "/skydns/services/"+m.UUID, bytes.NewBuffer(b))
	resp := httptest.NewRecorder()

	s.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatal("Failed to update TTL")
	}

	if serv, err := s.registry.GetUUID(m.UUID); err != nil || serv.TTL != 24 {
		t.Fatal("Failed to update TTL", err, serv.TTL)
	}
}

func TestUpdateTTLUnknownService(t *testing.T) {
	s := newTestServer("", 9560, 9561, "")
	defer s.Stop()

	m := msg.Service{
		UUID:        "54321",
		Name:        "TestService",
		Version:     "1.0.0",
		Region:      "Test",
		Host:        "localhost",
		Environment: "Production",
		Port:        9000,
		TTL:         4,
	}

	b, err := json.Marshal(m)

	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("PATCH", "/skydns/services/"+m.UUID, bytes.NewBuffer(b))
	resp := httptest.NewRecorder()

	s.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound || s.registry.Len() != 0 {
		t.Fatal("API should return 404 when updating unknown service")
	}
}

func TestGetService(t *testing.T) {
	s := newTestServer("", 9570, 9571, "")
	defer s.Stop()

	m := msg.Service{
		UUID:        "123",
		Name:        "TestService",
		Version:     "1.0.0",
		Region:      "Test",
		Host:        "localhost",
		Environment: "Production",
		Port:        9000,
		TTL:         4,
		Expires:     getExpirationTime(4),
	}

	s.registry.Add(m)

	req, _ := http.NewRequest("GET", "/skydns/services/"+m.UUID, nil)
	resp := httptest.NewRecorder()

	s.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatal("Failed to retrieve service")
	}

	m.TTL = 3 // TTL will be lower as time has passed
	expected, err := json.Marshal(m)

	if err != nil {
		t.Fatal(err)
	}

	// Newline is expected
	expected = append(expected, []byte("\n")...)

	if !bytes.Equal(resp.Body.Bytes(), expected) {
		t.Fatalf("Returned service is invalid. Expected %q but received %q", string(expected), resp.Body.String())
	}
}

func TestGetEnvironments(t *testing.T) {
	s := newTestServer("", 8500, 8501, "")
	defer s.Stop()

	for _, m := range services {
		s.registry.Add(m)
	}

	req, _ := http.NewRequest("GET", "/skydns/environments/", nil)
	resp := httptest.NewRecorder()

	s.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatal("Failed to retrieve environment list")
	}

	//server sends \n at the end, account for this
	expected := `{"Development":2,"Production":5}`
	expected = expected + "\n"

	if !bytes.Equal(resp.Body.Bytes(), []byte(expected)) {
		t.Fatal("Expected ", expected, " got %s", string(resp.Body.Bytes()))
	}

}

func TestGetRegions(t *testing.T) {
	s := newTestServer("", 8600, 8601, "")
	defer s.Stop()

	for _, m := range services {
		s.registry.Add(m)
	}

	req, _ := http.NewRequest("GET", "/skydns/regions/", nil)
	resp := httptest.NewRecorder()

	s.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatal("Failed to retrieve region list")
	}

	//server sends \n at the end, account for this
	expected := `{"Region1":3,"Region2":2,"Region3":2}`
	expected = expected + "\n"

	if !bytes.Equal(resp.Body.Bytes(), []byte(expected)) {
		t.Fatal("Expected ", expected, " got %s", string(resp.Body.Bytes()))
	}

}

func TestAuthenticationFailure(t *testing.T) {
	s := newTestServer("", 9610, 9611, "supersecretpassword")
	defer s.Stop()

	m := msg.Service{
		Name:        "TestService",
		Version:     "1.0.0",
		Region:      "Test",
		Host:        "localhost",
		Environment: "Production",
		Port:        9000,
		TTL:         4,
	}

	b, err := json.Marshal(m)

	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("PUT", "/skydns/services/123", bytes.NewBuffer(b))
	resp := httptest.NewRecorder()

	s.router.ServeHTTP(resp, req)
	if resp.Code != 403 {
		t.Fatal("Authentication should have failed and it worked.")
	}
}

func TestAuthenticationSuccess(t *testing.T) {
	secret := "myimportantsecret"
	s := newTestServer("", 9620, 9621, secret)
	defer s.Stop()

	m := msg.Service{
		Name:        "TestService",
		Version:     "1.0.0",
		Region:      "Test",
		Host:        "localhost",
		Environment: "Production",
		Port:        9000,
		TTL:         4,
	}

	b, err := json.Marshal(m)

	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("PUT", "/skydns/services/123", bytes.NewBuffer(b))
	req.Header.Set("Authorization", secret)
	resp := httptest.NewRecorder()

	s.router.ServeHTTP(resp, req)
	if resp.Code != 201 {
		t.Fatal("Auth Should have worked and it failed")
	}
}

var services = []msg.Service{
	msg.Service{
		UUID:        "100",
		Name:        "TestService",
		Version:     "1.0.0",
		Region:      "Region1",
		Host:        "server1",
		Environment: "Development",
		Port:        9000,
		TTL:         30,
		Expires:     getExpirationTime(30),
	},
	msg.Service{
		UUID:        "101",
		Name:        "TestService",
		Version:     "1.0.1",
		Region:      "Region1",
		Host:        "server2",
		Environment: "Production",
		Port:        9001,
		TTL:         31,
		Expires:     getExpirationTime(31),
	},
	msg.Service{
		UUID:        "102",
		Name:        "OtherService",
		Version:     "1.0.0",
		Region:      "Region2",
		Host:        "server3",
		Environment: "Production",
		Port:        9002,
		TTL:         32,
		Expires:     getExpirationTime(32),
	},
	msg.Service{
		UUID:        "103",
		Name:        "TestService",
		Version:     "1.0.1",
		Region:      "Region1",
		Host:        "server4",
		Environment: "Development",
		Port:        9003,
		TTL:         33,
		Expires:     getExpirationTime(33),
	},
	msg.Service{
		UUID:        "104",
		Name:        "TestService",
		Version:     "1.0.0",
		Region:      "Region3",
		Host:        "server5",
		Environment: "Production",
		Port:        9004,
		TTL:         34,
		Expires:     getExpirationTime(34),
	},
	msg.Service{
		UUID:        "105",
		Name:        "TestService",
		Version:     "1.0.0",
		Region:      "Region3",
		Host:        "server6",
		Environment: "Production",
		Port:        9005,
		TTL:         35,
		Expires:     getExpirationTime(35),
	},
	msg.Service{
		UUID:        "106",
		Name:        "OtherService",
		Version:     "1.0.0",
		Region:      "Region2",
		Host:        "server7",
		Environment: "Production",
		Port:        9006,
		TTL:         36,
		Expires:     getExpirationTime(36),
	},
}

type dnsTestCase struct {
	Question string
	Answer   []dns.SRV
}

var dnsTestCases = []dnsTestCase{
	// Generic Test
	dnsTestCase{
		Question: "testservice.production.skydns.local.",
		Answer: []dns.SRV{
			dns.SRV{
				Hdr: dns.RR_Header{
					Name:   "testservice.production.skydns.local.",
					Ttl:    30,
					Rrtype: dns.TypeSRV,
				},
				Priority: 10,
				Weight:   33,
				Target:   "server2.",
				Port:     9001,
			},
			dns.SRV{
				Hdr: dns.RR_Header{
					Name:   "testservice.production.skydns.local.",
					Ttl:    33,
					Rrtype: dns.TypeSRV,
				},
				Priority: 10,
				Weight:   33,
				Target:   "server5.",
				Port:     9004,
			},
			dns.SRV{
				Hdr: dns.RR_Header{
					Name:   "testservice.production.skydns.local.",
					Ttl:    34,
					Rrtype: dns.TypeSRV,
				},
				Priority: 10,
				Weight:   33,
				Target:   "server6.",
				Port:     9005,
			},
		},
	},

	// Region Priority Test
	dnsTestCase{
		Question: "region1.any.testservice.production.skydns.local.",
		Answer: []dns.SRV{
			dns.SRV{
				Hdr: dns.RR_Header{
					Name:   "region1.any.testservice.production.skydns.local.",
					Ttl:    30,
					Rrtype: dns.TypeSRV,
				},
				Priority: 10,
				Weight:   100,
				Target:   "server2.",
				Port:     9001,
			},
			dns.SRV{
				Hdr: dns.RR_Header{
					Name:   "region1.any.testservice.production.skydns.local.",
					Ttl:    33,
					Rrtype: dns.TypeSRV,
				},
				Priority: 20,
				Weight:   50,
				Target:   "server5.",
				Port:     9004,
			},
			dns.SRV{
				Hdr: dns.RR_Header{
					Name:   "region1.any.testservice.production.skydns.local.",
					Ttl:    34,
					Rrtype: dns.TypeSRV,
				},
				Priority: 20,
				Weight:   50,
				Target:   "server6.",
				Port:     9005,
			},
		},
	},
}

type servicesTest struct {
	query string
	count int
}

var serviceTestArray []servicesTest = []servicesTest{
	{"any", 7},
	{"production", 5},
	{"testservice.production", 3},
	{"region1.any.any.production", 1},
	{"region1.any.testservice.production", 1},
}

func TestGetServicesWithQueries(t *testing.T) {
	s := newTestServer("", 9590, 9591, "")
	defer s.Stop()

	for _, m := range services {
		s.registry.Add(m)
	}

	for _, st := range serviceTestArray {
		req, _ := http.NewRequest("GET", "/skydns/services/?query="+st.query, nil)
		resp := httptest.NewRecorder()
		s.router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatal("Failed To Retrieve Services")
		}
		var returnedServices []msg.Service
		err := json.Unmarshal(resp.Body.Bytes(), &returnedServices)
		if err != nil {
			t.Fatal("Failed to unmarshal response from server")
		}
		if len(returnedServices) != st.count {
			t.Fatal("Expected ", st.count, " got %d services", len(returnedServices))
		}

	}

}

func TestDNS(t *testing.T) {
	s := newTestServer("", 9580, 9581, "")
	defer s.Stop()

	for _, m := range services {
		s.registry.Add(m)
	}

	c := new(dns.Client)

	for _, tc := range dnsTestCases {
		m := new(dns.Msg)
		m.SetQuestion(tc.Question, dns.TypeSRV)
		resp, _, err := c.Exchange(m, "localhost:9580")

		if err != nil {
			t.Fatal(err)
		}

		if len(resp.Answer) != len(tc.Answer) {
			t.Fatalf("Response for %q contained %d results, %d expected", tc.Question, len(resp.Answer), len(tc.Answer))
		}

		for i, a := range resp.Answer {
			srv := a.(*dns.SRV)

			// Validate Header
			if srv.Hdr.Name != tc.Answer[i].Hdr.Name {
				t.Errorf("Answer %d should have a Header Name of %q, but has %q", i, tc.Answer[i].Hdr.Name, srv.Hdr.Name)
			}

			if srv.Hdr.Ttl != tc.Answer[i].Hdr.Ttl {
				t.Errorf("Answer %d should have a Header TTL of %d, but has %d", i, tc.Answer[i].Hdr.Ttl, srv.Hdr.Ttl)
			}

			if srv.Hdr.Rrtype != tc.Answer[i].Hdr.Rrtype {
				t.Errorf("Answer %d should have a Header Response Type of %d, but has %d", i, tc.Answer[i].Hdr.Rrtype, srv.Hdr.Rrtype)
			}

			// Validate Record
			if srv.Priority != tc.Answer[i].Priority {
				t.Errorf("Answer %d should have a Priority of %d, but has %d", i, tc.Answer[i].Priority, srv.Priority)
			}

			if srv.Weight != tc.Answer[i].Weight {
				t.Errorf("Answer %d should have a Weight of %d, but has %d", i, tc.Answer[i].Weight, srv.Weight)
			}

			if srv.Port != tc.Answer[i].Port {
				t.Errorf("Answer %d should have a Port of %d, but has %d", i, tc.Answer[i].Port, srv.Port)
			}

			if srv.Target != tc.Answer[i].Target {
				t.Errorf("Answer %d should have a Target of %q, but has %q", i, tc.Answer[i].Target, srv.Target)
			}
		}
	}
}

func TestDNSARecords(t *testing.T) {
	s := newTestServer("", 9600, 9601, "")
	defer s.Stop()

	c := new(dns.Client)

	m := new(dns.Msg)
	m.SetQuestion("skydns.local.", dns.TypeA)
	resp, _, err := c.Exchange(m, "localhost:9600")

	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Answer) != 1 {
		t.Fatal("Answer expected to have 2 A records but has", len(resp.Answer))
	}
}

func newTestServer(leader string, dnsPort int, httpPort int, secret string) *Server {
	members := make([]string, 0)

	p, _ := ioutil.TempDir("", "skydns-test-")
	if err := os.MkdirAll(p, 0644); err != nil {
		panic(err.Error())
	}

	if leader != "" {
		members = append(members, leader)
	}

	server := NewServer(members, "skydns.local", net.JoinHostPort("127.0.0.1", strconv.Itoa(dnsPort)), net.JoinHostPort("127.0.0.1", strconv.Itoa(httpPort)), p, 1*time.Second, 1*time.Second, secret)
	server.Start()

	return server
}
