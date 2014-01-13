package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"
	"time"

	"github.com/vito/garden/backend"
	"github.com/vito/garden/backend/fake_backend"
	"github.com/vito/garden/command_runner"
	"github.com/vito/garden/command_runner/remote_command_runner"
	"github.com/vito/garden/linux_backend"
	"github.com/vito/garden/linux_backend/container_pool"
	"github.com/vito/garden/linux_backend/network_pool"
	"github.com/vito/garden/linux_backend/port_pool"
	"github.com/vito/garden/linux_backend/quota_manager"
	"github.com/vito/garden/linux_backend/uid_pool"
	"github.com/vito/garden/server"
)

var socketFilePath = flag.String(
	"socket",
	"/tmp/warden.sock",
	"where to put the wardern server .sock file",
)

var snapshotsPath = flag.String(
	"snapshots",
	"",
	"directory in which to store container state to persist through restarts",
)

var backendName = flag.String(
	"backend",
	"linux",
	"which backend to use (linux or fake)",
)

var rootPath = flag.String(
	"root",
	"",
	"directory containing backend-specific scripts (i.e. ./linux/create.sh)",
)

var depotPath = flag.String(
	"depot",
	"",
	"directory in which to store containers",
)

var rootFSPath = flag.String(
	"rootfs",
	"",
	"directory of the rootfs for the containers",
)

var remoteHost = flag.String(
	"remoteHost",
	"",
	"machine to use for the Linux backend",
)

var remotePort = flag.Int(
	"remotePort",
	22,
	"SSH port of the remote machine",
)

var disableQuotas = flag.Bool(
	"disableQuotas",
	false,
	"disable disk quotas",
)

var containerGraceTime = flag.Int(
	"containerGraceTime",
	0,
	"time (in seconds) after which to destroy idle containers",
)

var debug = flag.Bool(
	"debug",
	false,
	"show low-level command output",
)

func main() {
	flag.Parse()

	maxProcs := runtime.NumCPU()
	prevMaxProcs := runtime.GOMAXPROCS(maxProcs)

	log.Println("set GOMAXPROCS to", maxProcs, "was", prevMaxProcs)

	var backend backend.Backend

	switch *backendName {
	case "linux":
		if *rootPath == "" {
			log.Fatalln("must specify -root with linux backend")
		}

		if *depotPath == "" {
			log.Fatalln("must specify -depot with linux backend")
		}

		if *rootFSPath == "" {
			log.Fatalln("must specify -rootfs with linux backend")
		}

		uidPool := uid_pool.New(10000, 256)

		_, ipNet, err := net.ParseCIDR("10.254.0.0/22")
		if err != nil {
			log.Fatalln("error parsing CIDR:", err)
		}

		networkPool := network_pool.New(ipNet)

		// TODO: base on ephemeral port range
		portPool := port_pool.New(61000, 6501)

		var runner command_runner.CommandRunner

		runner = command_runner.New(*debug)

		if *remoteHost != "" {
			runner = remote_command_runner.New(
				"root",
				*remoteHost,
				uint32(*remotePort),
				"/host",
				runner,
			)
		}

		quotaManager, err := quota_manager.New(*depotPath, *rootPath, runner)
		if err != nil {
			log.Fatalln("error creating quota manager:", err)
		}

		if *disableQuotas {
			quotaManager.Disable()
		}

		pool := container_pool.New(
			path.Join(*rootPath, "linux"),
			*depotPath,
			*rootFSPath,
			uidPool,
			networkPool,
			portPool,
			runner,
			quotaManager,
		)

		backend = linux_backend.New(pool, *snapshotsPath)
	case "fake":
		backend = fake_backend.New()
	}

	log.Println("setting up backend")

	err := backend.Setup()
	if err != nil {
		log.Fatalln("failed to set up backend:", err)
	}

	log.Println("starting server; listening on", *socketFilePath)

	graceTime := time.Duration(*containerGraceTime) * time.Second

	wardenServer := server.New(*socketFilePath, graceTime, backend)

	err = wardenServer.Start()
	if err != nil {
		log.Fatalln("failed to start:", err)
	}

	signals := make(chan os.Signal, 1)

	go func() {
		<-signals
		log.Println("stopping...")
		wardenServer.Stop()
		os.Exit(0)
	}()

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {}
}
