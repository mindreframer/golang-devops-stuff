package ostential
import (
	"ostential/assets"
	"ostential/view"

	"os"
	"fmt"
	"log"
	"net"
	"flag"
	"time"
	"sync"
	"bytes"
	"strings"
	"net/http"
	"path/filepath"

	"github.com/codegangsta/martini"
)

func parseaddr(bind_spec, defport string) (string, error) {
	if !strings.Contains(bind_spec, ":") {
		bind_spec = ":" + bind_spec
	}
	host, port, err := net.SplitHostPort(bind_spec)
	if err != nil {
		return "", err
	}
	if host == "*" {
		host = ""
	} else if port == "127" {
		host = "127.0.0.1"
		port = defport
	}
	if _, err = net.LookupPort("tcp", port); err != nil {
		if host != "" {
			return "", err
		}
		host = port
		port = defport
	}
	if err = os.Setenv("HOST", host); err != nil { return "", err }
	if err = os.Setenv("PORT", port); err != nil { return "", err }

	return host + ":" + port, nil
}

var ZEROTIME, _ = time.Parse("15:04:05", "00:00:00")
var opt_bindaddr string
const DEFPORT = "8050"
func init() {
	flag.StringVar(&opt_bindaddr, "b",    ":"+ DEFPORT, "Bind address")
	flag.StringVar(&opt_bindaddr, "bind", ":"+ DEFPORT, "Bind address")
}

type FlagError struct {
	error
}
func Listen() (net.Listener, error) {
	if !flag.Parsed() {
		flag.Parse()
	}
	bindaddr, err := parseaddr(opt_bindaddr, DEFPORT)
	if err != nil {
		return nil, FlagError{err}
	}
	listen, err := net.Listen("tcp", bindaddr)
	if err != nil {
		return nil, err
	}
	return listen, nil
}

func newModern() *Modern { // customized martini.Classic
	r := martini.NewRouter()
	m := martini.New()
	m.Use(martini.Recovery())
	m.Action(r.Handle)

	return &Modern{
		Martini: m,
		Router: r,
	}
}

func Serve(listen net.Listener, logfunc Logfunc, cb func(*Modern)) error {
	m := newModern() // as oppose to classic
	if cb != nil {
		cb(m)
	}

	logger := log.New(os.Stderr, "[ostent] ", 0)
	m.Map(logger) // log.Logger object

	m.Use(logfunc) // log middleware

	m.Use(assets_bindata())
	m.Use(view.BinTemplates_MartiniHandler())

	m.Get("/",   index)
	m.Get("/ws", slashws)

	addr := listen.Addr()
	if h, port, err := net.SplitHostPort(addr.String()); err == nil && h == "::" {
		// wildcard bind

		/* _, IP := NewInterfaces()
		logger.Printf("        http://%s", IP) // */
		addrs, err := net.InterfaceAddrs()
		if err == nil {
			for _, a := range addrs {
				ipnet, ok := a.(*net.IPNet)
				if !ok || strings.Contains(ipnet.IP.String(), ":") {
					continue // no IPv6 for now
				}
				logger.Printf("http://%s:%s", ipnet.IP.String(), port)
			}
		}
	} else {
		logger.Printf("http://%s", addr.String())
	}

	server := &http.Server{Addr: listen.Addr().String(), Handler: m}
	return server.Serve(listen)
}

func assets_bindata() martini.Handler {
	return func(res http.ResponseWriter, req *http.Request, log *log.Logger) {
		if req.Method != "GET" && req.Method != "HEAD" {
			return
		}
		path := req.URL.Path
		if path == "/" || path == "" || filepath.Ext(path) == ".go" { // cover the bindata.go
			return
		}
		if path[0] == '/' {
			path = path[1:]
		}
		text, err := assets.Asset(path)
		if err != nil {
			return
		}
		reader := bytes.NewReader(text)
		http.ServeContent(res, req, path, assets.ModTime(), reader)
	}
}

type Logfunc martini.Handler

var logOneLock sync.Mutex
var logged = map[string]bool{}
func LogOne(res http.ResponseWriter, req *http.Request, c martini.Context, logger *log.Logger) {
	start := time.Now()
	c.Next()

	rw := res.(martini.ResponseWriter)
	status := rw.Status()
	if status != 200 && status != 304 && req.URL.Path != "/ws" {
		logThis(start, res, req, logger)
		return
	}

	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		host = req.RemoteAddr
	}
	logOneLock.Lock()
	if _, ok := logged[host]; ok {
		logOneLock.Unlock()
		return
	}
	logged[host] = true
	logOneLock.Unlock()

	logger.Printf("%s\tRequested from %s; subsequent successful requests will not be logged\n", time.Now().Format("15:04:05"), host)
}

func LogAll(res http.ResponseWriter, req *http.Request, c martini.Context, logger *log.Logger) {
	start := time.Now()
	c.Next()
	logThis(start, res, req, logger)
}

func logThis(start time.Time, res http.ResponseWriter, req *http.Request, logger *log.Logger) {
	diff := time.Since(start)
	since := ZEROTIME.Add(diff).Format("5.0000s")

	rw := res.(martini.ResponseWriter)
	status := rw.Status()
	code := fmt.Sprintf("%d", status)
	if status != 200 {
		text := http.StatusText(status)
		if text != "" {
			code += fmt.Sprintf(" %s", text)
		}
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		host = req.RemoteAddr
	}

	logger.Printf("%s\t%s\t%s\t%v\t%s\t%s\n", start.Format("15:04:05"), host, since, code, req.Method, req.URL.Path)
}
