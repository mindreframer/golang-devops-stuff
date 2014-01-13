package gor

import (
	"encoding/json"
	"github.com/mattbaird/elastigo/api"
	"github.com/mattbaird/elastigo/core"
	"log"
	"net/http"
	"regexp"
	"time"
)

type ESUriErorr struct{}

func (e *ESUriErorr) Error() string {
	return "Wrong ElasticSearch URL format. Expected to be: host:port/index_name"
}

type ESPlugin struct {
	Active  bool
	ApiPort string
	Host    string
	Index   string
	indexor *core.BulkIndexer
	done    chan bool
}

type ESRequestResponse struct {
	ReqUrl               string         `json:"Req_URL"`
	ReqMethod            string         `json:"Req_Method"`
	ReqUserAgent         string         `json:"Req_User-Agent"`
	ReqAcceptLanguage    string         `json:"Req_Accept-Language,omitempty"`
	ReqAccept            string         `json:"Req_Accept,omitempty"`
	ReqAcceptEncoding    string         `json:"Req_Accept-Encoding,omitempty"`
	ReqIfModifiedSince   string         `json:"Req_If-Modified-Since,omitempty"`
	ReqConnection        string         `json:"Req_Connection,omitempty"`
	ReqCookies           []*http.Cookie `json:"Req_Cookies,omitempty"`
	RespStatus           string         `json:"Resp_Status"`
	RespStatusCode       int            `json:"Resp_Status-Code"`
	RespProto            string         `json:"Resp_Proto,omitempty"`
	RespContentLength    int64          `json:"Resp_Content-Length,omitempty"`
	RespContentType      string         `json:"Resp_Content-Type,omitempty"`
	RespTransferEncoding []string       `json:"Resp_Transfer-Encoding,omitempty"`
	RespContentEncoding  string         `json:"Resp_Content-Encoding,omitempty"`
	RespExpires          string         `json:"Resp_Expires,omitempty"`
	RespCacheControl     string         `json:"Resp_Cache-Control,omitempty"`
	RespVary             string         `json:"Resp_Vary,omitempty"`
	RespSetCookie        string         `json:"Resp_Set-Cookie,omitempty"`
	Rtt                  int64          `json:"RTT"`
	Timestamp            time.Time
}

// Parse ElasticSearch URI
//
// Proper format is: host:port/index_name
func parseURI(URI string) (err error, host string, port string, index string) {
	rURI := regexp.MustCompile("(.+):([0-9]+)/(.+)")
	match := rURI.FindAllStringSubmatch(URI, -1)

	if len(match) == 0 {
		err = new(ESUriErorr)
	} else {
		host = match[0][1]
		port = match[0][2]
		index = match[0][3]
	}

	return
}

func (p *ESPlugin) Init(URI string) {
	var err error

	err, p.Host, p.ApiPort, p.Index = parseURI(URI)

	if err != nil {
		log.Fatal("Can't initialize ElasticSearch plugin.", err)
	}

	api.Domain = p.Host
	api.Port = p.ApiPort

	p.indexor = core.NewBulkIndexerErrors(50, 60)
	p.done = make(chan bool)
	p.indexor.Run(p.done)

	// Only start the ErrorHandler goroutine when in verbose mode
	// no need to burn ressources otherwise
	// go p.ErrorHandler()

	log.Println("Initialized Elasticsearch Plugin")
	return
}

func (p *ESPlugin) IndexerShutdown() {
	p.done <- true
	return
}

func (p *ESPlugin) ErrorHandler() {
	for {
		errBuf := <-p.indexor.ErrorChannel
		log.Println(errBuf.Err)
	}
}

func (p *ESPlugin) RttDurationToMs(d time.Duration) int64 {
	sec := d / time.Second
	nsec := d % time.Second
	fl := float64(sec) + float64(nsec)*1e-6
	return int64(fl)
}

func (p *ESPlugin) ResponseAnalyze(req *http.Request, resp *http.Response, start, stop time.Time) {
	if resp == nil {
		// nil http response - skipped elasticsearch export for this request
		return
	}
	t := time.Now()
	rtt := p.RttDurationToMs(stop.Sub(start))

	esResp := ESRequestResponse{
		ReqUrl:               req.URL.String(),
		ReqMethod:            req.Method,
		ReqUserAgent:         req.UserAgent(),
		ReqAcceptLanguage:    req.Header.Get("Accept-Language"),
		ReqAccept:            req.Header.Get("Accept"),
		ReqAcceptEncoding:    req.Header.Get("Accept-Encoding"),
		ReqIfModifiedSince:   req.Header.Get("If-Modified-Since"),
		ReqConnection:        req.Header.Get("Connection"),
		ReqCookies:           req.Cookies(),
		RespStatus:           resp.Status,
		RespStatusCode:       resp.StatusCode,
		RespProto:            resp.Proto,
		RespContentLength:    resp.ContentLength,
		RespContentType:      resp.Header.Get("Content-Type"),
		RespTransferEncoding: resp.TransferEncoding,
		RespContentEncoding:  resp.Header.Get("Content-Encoding"),
		RespExpires:          resp.Header.Get("Expires"),
		RespCacheControl:     resp.Header.Get("Cache-Control"),
		RespVary:             resp.Header.Get("Vary"),
		RespSetCookie:        resp.Header.Get("Set-Cookie"),
		Rtt:                  rtt,
		Timestamp:            t,
	}
	j, err := json.Marshal(&esResp)
	if err != nil {
		log.Println(err)
	} else {
		p.indexor.Index(p.Index, "RequestResponse", "", "", &t, j)
	}
	return
}
