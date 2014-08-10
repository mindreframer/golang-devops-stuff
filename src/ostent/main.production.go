// +build production

package main
import (
	"libostent"

	"os"
	"net"
	"log"
	"fmt"
	"flag"
	"time"
 	"syscall"
	"runtime"
	"strings"
	"net/url"
	"net/http"
	"math/rand"
	"path/filepath"

	"github.com/rcrowley/goagain"

	"github.com/inconshreveable/go-update"
)

func init() {
// 	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	log.SetPrefix(fmt.Sprintf("[%d] ", os.Getpid()))
}

func upgrade_loop() {
	const dlimit = time.Hour
	delta := time.Duration(rand.Int63n(int64(dlimit)))
	for {
		select {
		case <-time.After(time.Hour * 1 + delta): // 1.5 +- 0.5 h
			if upgrade_once(true) {
				break
			}
		}
	}
}

func newer_version() string {
	// 1. https://github.com/rzab/ostent/releases/latest # redirects, NOT followed
	// 2. https://github.com/rzab/ostent/releases/vX.Y.Z # Redirect location
	// 3. return "vX.Y.Z" # basename of the location

	type redirected struct {
		error
		url url.URL
	}
	checkRedirect := func(req *http.Request, _via []*http.Request) error {
		return redirected{url: *req.URL,}
	}

	client := &http.Client{CheckRedirect: checkRedirect,}
	resp, err := client.Get("https://github.com/rzab/ostent/releases/latest")
	if err == nil {
		resp.Body.Close()
		return ""
	}
	urlerr, ok := err.(*url.Error)
	if !ok {
		fmt.Fprintln(os.Stderr, err)
		return ""
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	redir, ok := urlerr.Err.(redirected)
	if !ok {
		fmt.Fprintln(os.Stderr, err)
		return ""
	}
	return filepath.Base(redir.url.Path)
}

func upgrade_once(kill bool) bool {

	mach := runtime.GOARCH
	if mach == "amd64" {
		mach = "x86_64"
	} else if mach == "386" {
		mach = "i686"
	}
	new_version := newer_version()
	if new_version == "" || new_version == "v"+ ostent.VERSION {
		return false
	}
// 	url := fmt.Sprintf("https://ostrost.com"+ "/ostent/releases/%s/%s %s/newer",    ostent.VERSION, strings.Title(runtime.GOOS), mach) // before v0.1.3
	url := fmt.Sprintf("https://github.com/rzab/ostent/releases/download/%s/%s.%s", new_version,  strings.Title(runtime.GOOS), mach)

	err, _ := update.New().FromUrl(url) // , snderr
	if err != nil ||  err != nil {
		// log.Printf("Upgrade failed: %v|%v\n", err, snderr)
		return false
	}
	log.Println("Successfull UPGRADE, relaunching")
	if kill {
		syscall.Kill(os.Getpid(), syscall.SIGUSR2)
	}
	return true
}

func main() {
	upgradelater := flag.Bool("upgradelater", false, "Upgrade later")

	flag.Parse()

	had_upgrade := false
	if !*upgradelater && os.Getenv("GOAGAIN_PPID") == "" { // not after gone again
		log.Println("Initial check for upgrades; run with -ugradelater to disable")
		had_upgrade = upgrade_once(false)
	}

	if !had_upgrade { // start the background routine unless just had an upgrade and gonna relaunch anyway
		go ostent.Loop()
		// go ostent.CollectdLoop()
	}

	listen, err := goagain.Listener()
	if err != nil {
		listen, err = net.Listen("tcp", ostent.OstentBindFlag.String())
		if err != nil {
			log.Fatalln(err)
		}

		if had_upgrade { // goagain
			go func() {
				time.Sleep(time.Second) // not before goagain.Wait
				syscall.Kill(os.Getpid(), syscall.SIGUSR2)
				// goagain.ForkExec(listen)
			}()
		} else {
			go upgrade_loop()
			go ostent.Serve(listen, true, nil)
		}

	} else {
		go upgrade_loop()
		go ostent.Serve(listen, true, nil)

		if err := goagain.Kill(); err != nil {
			log.Fatalln(err)
		}
	}

	if _, err := goagain.Wait(listen); err != nil { // signals before won't be catched
		log.Fatalln(err)
	}

	// shutting down

	if ostent.Connections.Reload() {
		time.Sleep(time.Second)
	} // */

	if err := listen.Close(); err != nil {
		log.Fatalln(err)
	}
	time.Sleep(time.Second)
}
