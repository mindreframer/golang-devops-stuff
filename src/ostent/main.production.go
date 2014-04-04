// +build production

package main
import (
	"ostential"

	"os"
	"log"
	"fmt"
	"flag"
	"time"
 	"syscall"
	"runtime"
	"strings"
	"math/rand"

	"github.com/codegangsta/martini"
	"github.com/rcrowley/goagain"

	"github.com/inconshreveable/go-update"
)

func init() {
// 	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	log.SetPrefix(fmt.Sprintf("[%d] ", os.Getpid()))
}

func update_loop() {
	const dlimit = time.Hour
	delta := time.Duration(rand.Int63n(int64(dlimit)))
	for {
		select {
		case <-time.After(time.Hour * 1 + delta): // 1.5 +- 0.5 h
			if update_once(true) {
				break
			}
		}
	}
}

const update_from = "0.1.1"
func update_once(kill bool) bool {

	host := "https://OSTROST.COM"

	mach := runtime.GOARCH
	if mach == "amd64" {
		mach = "x86_64"
	} else if mach == "386" {
		mach = "i686"
	}
	url := fmt.Sprintf("%s/ostent/releases/%s/%s %s/newer", host, update_from, strings.Title(runtime.GOOS), mach)

	err, _ := update.FromUrl(url) // , snderr
	if err != nil ||  err != nil {
		// log.Printf("Update failed: %v|%v\n", err, snderr)
		return false
	}
	log.Println("Successfull UPDATE, relaunching")
	if kill {
		syscall.Kill(os.Getpid(), syscall.SIGUSR2)
	}
	return true
}

func main() {
	updatelater := flag.Bool("updatelater", false, "Update later")
	flag.Parse()

	had_update := false
	if !*updatelater && os.Getenv("GOAGAIN_PPID") == "" { // not after gone again
		log.Println("Initial check for updates; run with -updatelater to disable")
		had_update = update_once(false)
	}

	martini.Env = martini.Prod
	listen, err := goagain.Listener()
	if err != nil {
		listen, err = ostential.Listen()
		if err != nil {
			if _, ok := err.(ostential.FlagError); ok {
				flag.Usage()
				os.Exit(2)
				return
			}
			log.Fatalln(err)
		}

		if had_update { // goagain
			go func() {
				time.Sleep(time.Second) // not before goagain.Wait
				syscall.Kill(os.Getpid(), syscall.SIGUSR2)
				// goagain.ForkExec(listen)
			}()
		} else {
			go update_loop()
			go ostential.Serve(listen, ostential.LogOne, nil)
		}

	} else {
		go update_loop()
		go ostential.Serve(listen, ostential.LogOne, nil)

		if err := goagain.Kill(); err != nil {
			log.Fatalln(err)
		}
	}

	if _, err := goagain.Wait(listen); err != nil { // signals before won't be catched
		log.Fatalln(err)
	}
	if err := listen.Close(); err != nil {
		log.Fatalln(err)
	}
	time.Sleep(time.Second)
}
