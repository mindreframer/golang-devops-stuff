package goshare

import (
  "fmt"
  "flag"
  "log"
  "os"
  "runtime"
  "runtime/pprof"
  "time"

  "github.com/jmhodges/levigo"
  abkleveldb "github.com/abhishekkr/levigoNS/leveldb"
)

var (
  db *levigo.DB
  dbpath      = flag.String("dbpath", "/tmp/GO.DB", "the path to DB")
  httpuri     = flag.String("uri", "0.0.0.0", "what Port to Run HTTP Server at")
  httpport    = flag.Int("port", 9999, "what Port to Run HTTP Server at")
  req_port    = flag.Int("req-port", 9797, "what PORT to run ZMQ REQ at")
  rep_port    = flag.Int("rep-port", 9898, "what PORT to run ZMQ REP at")
  cpuprofile  = flag.String("cpuprofile", "", "write cpu profile to file")
)

func banner(){
  fmt.Println("**************************************************")
  fmt.Println("  ___  ____      ___        __   _   __")
  fmt.Println("  |    |  |      |    |  | /  \\ | ) |")
  fmt.Println("  | =| |  |  =~  |==| |==| |==| |=  |=")
  fmt.Println("  |__| |__|      ___| |  | |  | | \\ |__")
  fmt.Println("")
  fmt.Println("**************************************************")
}

func do_you_wanna_continue(){
  var input string
  for {
    fmt.Println("Do you wanna exit. (yes|no):\n\n")

    fmt.Scanf("%s", &input)

    if input == "yes" || input == "y" { break }
  }
}

func GoShare(){
  banner()
  runtime.GOMAXPROCS(runtime.NumCPU())

  flag.Parse()
  db = abkleveldb.CreateDB(*dbpath)
  if *cpuprofile != "" {
    f, err := os.Create(*cpuprofile)
    if err != nil {
      log.Fatal(err)
    }
    pprof.StartCPUProfile(f)
    go func() {
      time.Sleep(100 * time.Second)
      pprof.StopCPUProfile()
    }()
  }

  // need to go CHAN passing msg to leveldb and back
  go GoShareHTTP(*httpuri, *httpport)
  go GoShareZMQ(*req_port, *rep_port)

  do_you_wanna_continue()
}
