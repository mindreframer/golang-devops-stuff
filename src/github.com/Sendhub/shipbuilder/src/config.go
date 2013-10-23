package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sendhub/logserver"
	"launchpad.net/goamz/aws"
)

type (
	Application struct {
		Name        string
		Domains     []string
		BuildPack   string
		Environment map[string]string
		Processes   map[string]int
		LastDeploy  string
		Maintenance bool
		Drains      []string
	}
	Node struct {
		Host string
	}
	Config struct {
		LoadBalancers []string
		Nodes         []*Node
		Port          int
		GitRoot       string
		LxcRoot       string
		Applications  []*Application
	}
)

const (
	APP_DIR                            = "/app"
	ENV_DIR                            = APP_DIR + "/env"
	LXC_DIR                            = "/var/lib/lxc"
	DIRECTORY                          = "/mnt/build"
	BINARY                             = "shipbuilder"
	EXE                                = DIRECTORY + "/" + BINARY
	CONFIG                             = DIRECTORY + "/config.json"
	GIT_DIRECTORY                      = "/git"
	DEFAULT_NODE_USERNAME              = "ubuntu"
	VERSION                            = "0.1.4"
	NODE_SYNC_TIMEOUT_SECONDS          = 180
	DYNO_START_TIMEOUT_SECONDS         = 120
	LOAD_BALANCER_SYNC_TIMEOUT_SECONDS = 45
	DEPLOY_TIMEOUT_SECONDS             = 240
	STATUS_MONITOR_INTERVAL_SECONDS    = 15
	DEFAULT_SSH_PARAMETERS             = "-o StrictHostKeyChecking=no -o BatchMode=yes -o ConnectTimeout=30" // NB: Notice 30s connect timeout.
)

// LDFLAGS can be specified by compiling with `-ldflags '-X main.defaultSshHost=.. ...'`.
var (
	build                     string
	defaultHaProxyStats       bool
	defaultHaProxyCredentials string
	defaultAwsKey             string
	defaultAwsSecret          string
	defaultAwsRegion          string
	defaultS3BucketName       string
	defaultSshHost            string
	defaultSshKey             string
	defaultLxcFs              string
	defaultZfsPool            string
)

// Global configuration.
var (
	sshHost      = OverridableByEnv("SB_SSH_HOST", defaultSshHost)
	sshKey       = OverridableByEnv("SB_SSH_KEY", defaultSshKey)
	awsKey       = OverridableByEnv("SB_AWS_KEY", defaultAwsKey)
	awsSecret    = OverridableByEnv("SB_AWS_SECRET", defaultAwsSecret)
	awsRegion    = GetAwsRegion("SB_AWS_REGION", defaultAwsRegion)
	s3BucketName = OverridableByEnv("SB_S3_BUCKET", defaultS3BucketName)
	lxcFs        = OverridableByEnv("LXC_FS", defaultLxcFs)
	zfsPool      = OverridableByEnv("ZFS_POOL", defaultZfsPool)
)

var (
	defaultSshParametersList = strings.Split(DEFAULT_SSH_PARAMETERS, " ")
	configLock               sync.Mutex
	syncLoadBalancerLock     sync.Mutex
)

func (this *Application) LxcDir() string {
	return LXC_DIR + "/" + this.Name
}
func (this *Application) RootFsDir() string {
	return LXC_DIR + "/" + this.Name + "/rootfs"
}
func (this *Application) AppDir() string {
	return this.RootFsDir() + APP_DIR
}
func (this *Application) SrcDir() string {
	return this.AppDir() + "/src"
}
func (this *Application) LocalAppDir() string {
	return APP_DIR
}
func (this *Application) LocalSrcDir() string {
	return APP_DIR + "/src"
}
func (this *Application) BaseContainerName() string {
	return "base-" + this.BuildPack
}
func (this *Application) GitDir() string {
	return GIT_DIRECTORY + "/" + this.Name
}

// Get total requested number of Dynos (based on Processes).
func (this *Application) TotalRequestedDynos() int {
	n := 0
	for _, value := range this.Processes {
		if value > 0 { // Ensure negative values are never added.
			n += value
		}
	}
	return n
}

// Entire maintenance page URL (e.g. "http://example.com/static/maintenance.html").
func (this *Application) MaintenancePageUrl() string {
	maintenanceUrl, ok := this.Environment["MAINTENANCE_PAGE_URL"]
	if ok {
		return maintenanceUrl
	}
	// Fall through to searching for a universal maintenance page URL in an environment variable, and
	// defaulting to a potentially useful page.
	var firstDomain string
	if len(this.Domains) > 0 {
		firstDomain = this.Domains[0]
	} else {
		firstDomain = "example.com"
	}
	return ConfigFromEnv("SB_DEFAULT_MAINTENANCE_URL", "http://www.downforeveryoneorjustme.com/"+firstDomain)
}

// Maintenance page URL path. (e.g. "/static/maintenance.html").
func (this *Application) MaintenancePageFullPath() string {
	maintenanceUrl := this.MaintenancePageUrl()
	if len(maintenanceUrl) < 3 {
		fmt.Printf("error :: Application.MaintenancePageFullPath :: url too short: '%v'\n", maintenanceUrl)
		return maintenanceUrl
	}
	u, err := url.Parse(maintenanceUrl)
	if err != nil {
		fmt.Printf("error :: Application.MaintenancePageFullPath :: %v: '%v'\n", err, maintenanceUrl)
		return "/"
	}
	return u.Path
}

// Maintenance page URL without the document name (e.g. "/static/).
func (this *Application) MaintenancePageBasePath() string {
	path := this.MaintenancePageFullPath()
	i := strings.LastIndex(path, "/")
	if i == -1 || len(path) == 1 {
		return path
	}
	return path[0:i]
}

// Maintenance page URL domain-name. (e.g. "example.com").
func (this *Application) MaintenancePageDomain() string {
	maintenanceUrl := this.MaintenancePageUrl()
	if len(maintenanceUrl) < 3 {
		fmt.Printf("error :: Application.MaintenancePageDomain :: url too short: '%v'\n", maintenanceUrl)
		return maintenanceUrl
	}
	u, err := url.Parse(maintenanceUrl)
	if err != nil {
		fmt.Printf("error :: Application.MaintenancePageDomain :: %v: '%v'\n", err, maintenanceUrl)
		return "domain-parse-failed"
	}
	return u.Host
}

func (this *Application) NextVersion() (string, error) {
	if this.LastDeploy == "" {
		return "v1", nil
	}
	current, err := strconv.Atoi(this.LastDeploy[1:])
	if err != nil {
		return "", err
	}
	return "v" + strconv.Itoa(current+1), nil
}
func (this *Application) CalcPreviousVersion() (string, error) {
	if this.LastDeploy == "" || this.LastDeploy == "v1" {
		return "", nil
	}
	v, err := strconv.Atoi(this.LastDeploy[1:])
	if err != nil {
		return "", err
	}
	return "v" + strconv.Itoa(v-1), nil
}
func (this *Application) CreateBaseContainerIfMissing(e *Executor) error {
	if !e.ContainerExists(this.Name) {
		return e.CloneContainer(this.BaseContainerName(), this.Name)
	}
	return nil
}

func (this *Server) IncrementAppVersion(app *Application) (*Application, *Config, error) {
	var updatedApp *Application
	var updatedCfg *Config

	err := this.WithPersistentApplication(app.Name, func(app *Application, cfg *Config) error {
		nextVersion, err := app.NextVersion()
		fmt.Printf("NEXT VERSION OF %v -> %v\n", app.Name, nextVersion)
		if err != nil {
			return err
		}
		app.LastDeploy = nextVersion
		updatedApp = app
		updatedCfg = cfg
		return nil
	})
	return updatedApp, updatedCfg, err
}

// Only to be invoked by safe locking getters/setters, never externally!!!
func (this *Server) getConfig(lock bool) (*Config, error) {
	if lock {
		configLock.Lock()
		defer configLock.Unlock()
	}

	var config Config

	f, err := os.Open(CONFIG)
	if err == nil {
		defer f.Close()
		err = json.NewDecoder(f).Decode(&config)
		if err != nil {
			return nil, err
		}
	}

	if config.Applications == nil {
		config.Applications = []*Application{}
	}

	if config.LoadBalancers == nil {
		config.LoadBalancers = []string{}
	}
	if config.Nodes == nil {
		config.Nodes = []*Node{}
	}

	return &config, nil
}

// IMPORTANT: Only to be invoked by `WithPersistentConfig`.
func (this *Server) writeConfig(config *Config) error {
	f, err := os.Create(CONFIG)
	if err != nil {
		return err
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(config)
	if err != nil {
		return err
	}
	return nil
}

// Obtains the config lock, then applies the passed function which can mutate the config, then writes out the changes.
func (this *Server) WithPersistentConfig(fn func(*Config) error) error {
	configLock.Lock()
	defer configLock.Unlock()

	cfg, err := this.getConfig(false)
	if err != nil {
		return err
	}
	err = fn(cfg)
	if err != nil {
		return err
	}
	err = this.writeConfig(cfg)
	if err != nil {
		return err
	}
	return nil
}

// Reads the config and invokes the passed function with it.  Does not store any config changes.
func (this *Server) WithConfig(fn func(*Config) error) error {
	cfg, err := this.getConfig(true)
	if err != nil {
		return err
	}
	err = fn(cfg)
	if err != nil {
		return err
	}
	return nil
}

func (this *Server) WithPersistentApplication(name string, fn func(*Application, *Config) error) error {
	return this.WithPersistentConfig(func(cfg *Config) error {
		for _, app := range cfg.Applications {
			if app.Name == name {
				return fn(app, cfg)
			}
		}
		return fmt.Errorf("Unknown application: %v", name)
	})
}

func (this *Server) WithApplication(name string, fn func(*Application, *Config) error) error {
	return this.WithConfig(func(cfg *Config) error {
		for _, app := range cfg.Applications {
			if app.Name == name {
				return fn(app, cfg)
			}
		}
		return fmt.Errorf("Unknown application: %v", name)
	})
}

// Returns the ShipBuilder log server ip:port to send HAProxy UDP logs to.
// Autmatically takes care of transforming ssh hostname into just a hostname.
func (this *Server) ResolveLogServerIpAndPort() (string, error) {
	hostname := sshHost[int(math.Max(float64(strings.Index(sshHost, "@")+1), 0)):]
	ipAddr, err := net.ResolveIPAddr("ip", hostname)
	if err != nil {
		return "", err
	}
	ip := ipAddr.IP.String()
	port := strconv.Itoa(log.Port)
	return ip + ":" + port, nil
}

func (this *Server) SyncLoadBalancers(e *Executor, addDynos []Dyno, removeDynos []Dyno) error {
	syncLoadBalancerLock.Lock()
	defer syncLoadBalancerLock.Unlock()

	cfg, err := this.getConfig(true)
	if err != nil {
		return err
	}

	logServerIpAndPort, err := this.ResolveLogServerIpAndPort()
	if err != nil {
		return err
	}

	type Server struct {
		Host string
		Port int
	}
	type App struct {
		Name                    string
		Domains                 []string
		Servers                 []*Server
		Maintenance             bool
		MaintenancePageFullPath string
		MaintenancePageBasePath string
		MaintenancePageDomain   string
	}
	type Lb struct {
		LogServerIpAndPort  string // ShipBuilder server ip:port to send HAProxy UDP logs to.
		Applications        []*App
		LoadBalancers       []string
		HaProxyStatsEnabled bool
		HaProxyCredentials  string
	}
	lb := &Lb{
		LogServerIpAndPort:  logServerIpAndPort,
		Applications:        []*App{},
		LoadBalancers:       cfg.LoadBalancers,
		HaProxyStatsEnabled: HaProxyStatsEnabled(),
		HaProxyCredentials:  HaProxyCredentials(),
	}

	for _, app := range cfg.Applications {
		a := &App{
			Name:                    app.Name,
			Domains:                 app.Domains,
			Servers:                 []*Server{},
			Maintenance:             app.Maintenance,
			MaintenancePageFullPath: app.MaintenancePageFullPath(),
			MaintenancePageBasePath: app.MaintenancePageBasePath(),
			MaintenancePageDomain:   app.MaintenancePageDomain(),
		}
		for proc, _ := range app.Processes {
			if proc == "web" {
				// Find and don't add `removeDynos`.
				runningDynos, err := this.GetRunningDynos(app.Name, proc)
				if err != nil {
					return err
				}
				for _, dyno := range runningDynos {
					found := false
					for _, remove := range removeDynos {
						if remove == dyno {
							found = true
							break
						}
					}
					if found {
						continue
					}
					port, err := strconv.Atoi(dyno.Port)
					if err != nil {
						return err
					}
					a.Servers = append(a.Servers, &Server{
						Host: dyno.Host,
						Port: port,
					})
				}
				// Add `addDynos` if type is "web" and it matches the current application and process.
				for _, addDyno := range addDynos {
					if addDyno.Application == app.Name && addDyno.Process == proc {
						port, err := strconv.Atoi(addDyno.Port)
						if err != nil {
							return err
						}

						candidateServer := &Server{
							Host: addDyno.Host,
							Port: port,
						}

						// Check if the server has already been added.
						alreadyAdded := false
						for _, existingServer := range a.Servers {
							if existingServer == candidateServer {
								alreadyAdded = true
							}
						}
						if !alreadyAdded {
							a.Servers = append(a.Servers, candidateServer)
						}
					}
				}
			}
		}
		lb.Applications = append(lb.Applications, a)
	}

	// Save it to the load balancer
	f, err := os.OpenFile("/tmp/haproxy.cfg", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0x666)
	if err != nil {
		return err
	}
	defer os.Remove("/tmp/haproxy.cfg")
	err = HAPROXY_CONFIG.Execute(f, lb)
	f.Close()
	if err != nil {
		return err
	}

	type LbSyncResult struct {
		lbHost string
		err    error
	}

	syncChannel := make(chan LbSyncResult)
	for _, lb := range cfg.LoadBalancers {
		go func(lb string) {
			c := make(chan error, 1)
			go func() {
				err := e.Run("rsync",
					"-azve", "ssh "+DEFAULT_SSH_PARAMETERS,
					"/tmp/haproxy.cfg", "root@"+lb+":/etc/haproxy/",
				)
				if err != nil {
					c <- err
					return
				}
				err = e.Run("ssh", DEFAULT_NODE_USERNAME+"@"+lb,
					`sudo /bin/bash -c 'if [ "$(sudo service haproxy status)" = "haproxy not running." ]; then sudo service haproxy start; else sudo service haproxy reload; fi'`,
				)
				if err != nil {
					c <- err
					return
				}
				c <- nil
			}()
			go func() {
				time.Sleep(LOAD_BALANCER_SYNC_TIMEOUT_SECONDS * time.Second)
				c <- fmt.Errorf("LB sync operation to '%v' timed out after %v seconds", lb, LOAD_BALANCER_SYNC_TIMEOUT_SECONDS)
			}()
			// Block until chan has something, at which point syncChannel will be notified.
			syncChannel <- LbSyncResult{lb, <-c}
		}(lb)
	}

	nLoadBalancers := len(cfg.LoadBalancers)
	errors := []error{}
	for i := 1; i <= nLoadBalancers; i++ {
		syncResult := <-syncChannel
		if syncResult.err != nil {
			errors = append(errors, syncResult.err)
		}
		fmt.Fprintf(e.logger, "%v/%v load-balancer sync finished (%v succeeded, %v failed, %v outstanding)\n", i, nLoadBalancers, i-len(errors), len(errors), nLoadBalancers-i)
	}

	// If all LB updates failed, abort with error.
	if nLoadBalancers > 0 && len(errors) == nLoadBalancers {
		err = fmt.Errorf("error: all load-balancer updates failed: %v", errors)
		fmt.Fprintf(e.logger, "%v", err)
		return err
	}

	// Uddate `currentLoadBalancerConfig` with updated HAProxy configuration.
	cfgBuffer := bytes.Buffer{}
	err = HAPROXY_CONFIG.Execute(&cfgBuffer, lb)
	if err != nil {
		return err
	}
	this.currentLoadBalancerConfig = cfgBuffer.String()

	// Pause briefly to ensure HAProxy has time to complete it's reload.
	time.Sleep(time.Second * 1)

	return nil
}

// The first time this method is invoked the current config will read from a load-balancer, if one is available.
// Subsequent invocations will use the current version.
// After a deployment, the `SyncLoadBalancers` method automatically updates `currentLoadBalancerConfig`.
func (this *Server) GetActiveLoadBalancerConfig() (string, error) {
	if len(this.currentLoadBalancerConfig) == 0 {
		cfg, err := this.getConfig(true)
		if err != nil {
			return this.currentLoadBalancerConfig, err
		}
		if len(cfg.LoadBalancers) == 0 {
			return this.currentLoadBalancerConfig, fmt.Errorf("There are currently no load-balancers configured to pull LB config from")
		}

		syncLoadBalancerLock.Lock()
		defer syncLoadBalancerLock.Unlock()
		this.currentLoadBalancerConfig, err = RemoteCommand(cfg.LoadBalancers[0], "sudo cat /etc/haproxy/haproxy.cfg")
		if err != nil {
			return this.currentLoadBalancerConfig, err
		}
	} else {
		syncLoadBalancerLock.Lock()
		defer syncLoadBalancerLock.Unlock()
	}
	return this.currentLoadBalancerConfig, nil
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func MkdirIfNotExists(path string, perm os.FileMode) error {
	exists, err := PathExists(path)
	if err != nil {
		return err
	}
	if !exists {
		err = os.Mkdir(path, perm)
		if err != nil {
			return err
		}
	}
	return nil
}

func ConfigFromEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return value
}

func OverridableByEnv(key string, ldflagsValue string) string {
	envValue := os.Getenv(key)
	if len(envValue) > 0 {
		fmt.Printf("info: environmental override detected for %v: %v\n", key, envValue)
		return envValue
	}
	if len(ldflagsValue) == 0 {
		fmt.Printf("fatal: missing required configuration value for '%v'\n", key)
		os.Exit(1)
	}
	//fmt.Printf("info: ldflags value detected for %v: %v\n", key, ldflagsValue)
	return ldflagsValue
}

// Validate that the configured key exists in the provided options.
func GetAwsRegion(key string, ldflagsValue string) aws.Region {
	regionKey := OverridableByEnv(key, ldflagsValue)
	region, ok := aws.Regions[regionKey]
	if !ok {
		validRegions := []string{}
		for _, r := range aws.Regions {
			validRegions = append(validRegions, r.Name)
		}
		fmt.Printf("fatal: invalid option '%v' for parameter '%v', acceptable values are: %v\n", regionKey, key, validRegions)
		os.Exit(1)
	}
	return region
}

func GetSystemIp() string {
	name, err := os.Hostname()
	if err != nil {
		fmt.Printf("error: GetSystemIp: Oops-1: %v\n", err)
	} else {
		addrs, err := net.LookupHost(name)
		if err != nil {
			fmt.Printf("error: GetSystemIp: Oops-2: %v\n", err)
		} else if len(addrs) > 0 {
			return addrs[0]
		}
	}
	fmt.Printf("warning: GetSystemIp: system address discovery failed, defaulting to '127.0.0.1'\n")
	return "127.0.0.1"
}

func HaProxyStatsEnabled() bool {
	maybeValue := strings.TrimSpace(strings.ToLower(os.Getenv("SB_HAPROXY_STATS")))
	if maybeValue != "" {
		return maybeValue == "1" || maybeValue == "true" || maybeValue == "yes"
	}
	return defaultHaProxyStats
}

func HaProxyCredentials() string {
	maybeValue := os.Getenv("SB_HAPROXY_CREDENTIALS")
	if len(maybeValue) > 2 && strings.Contains(maybeValue, ":") {
		return maybeValue
	}
	if defaultHaProxyCredentials != "" && len(defaultHaProxyCredentials) > 2 && strings.Contains(defaultHaProxyCredentials, ":") {
		return defaultHaProxyCredentials
	}
	return "admin:password"
}
