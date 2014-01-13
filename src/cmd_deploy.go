package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type (
	// `ScalingOnly`: Flag to indicate whether this is a new release or a scaling activity.
	Deployment struct {
		StartedTs   time.Time
		Server      *Server
		Logger      io.Writer
		Application *Application
		Config      *Config
		Revision    string
		Version     string
		ScalingOnly bool
		err         error
	}
	DeployLock struct {
		numStarted  int
		numFinished int
		mutex       sync.Mutex
	}
)

// Notify the DeployLock of a newly started deployment.
func (this *DeployLock) start() {
	this.mutex.Lock()
	this.numStarted++
	this.mutex.Unlock()
}

// Mark a deployment as completed.
func (this *DeployLock) finish() {
	this.mutex.Lock()
	this.numFinished++
	this.mutex.Unlock()
}

// Obtain the current number of started deploys.  Used as a marker by the
// Dyno cleanup system to protect against taking action with stale data.
func (this *DeployLock) value() int {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return this.numStarted
}

// Return true if and only if no deploys are in progress and if a possibly
// out-of-date value matches the current DeployLock.numStarted value.
func (this *DeployLock) validateLatest(value int) bool {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return this.numStarted == this.numFinished && this.numStarted == value
}

// Keep track of deployment run count to avoid cleanup operating on stale data.
var deployLock = DeployLock{numStarted: 0, numFinished: 0}

func (this *Deployment) createContainer() error {
	dimLogger := NewFormatter(this.Logger, DIM)
	titleLogger := NewFormatter(this.Logger, GREEN)

	e := Executor{dimLogger}

	fmt.Fprintf(titleLogger, "Creating container\n")

	// If there's not already a container.
	_, err := os.Stat(this.Application.RootFsDir())
	if err != nil {
		// Clone the base application.
		this.err = e.CloneContainer("base-"+this.Application.BuildPack, this.Application.Name)
		if this.err != nil {
			return this.err
		}
	}

	e.BashCmd("rm -rf " + this.Application.AppDir())
	e.BashCmd("mkdir -p " + this.Application.SrcDir())
	// Copy the binary into the container.
	this.err = e.BashCmd("cp " + EXE + " " + this.Application.AppDir() + "/" + BINARY)
	if this.err != nil {
		return this.err
	}
	// Export the source to the container.
	this.err = e.BashCmd("git clone " + this.Application.GitDir() + " " + this.Application.SrcDir())
	if this.err != nil {
		return this.err
	}
	// Checkout the given revision.
	this.err = e.BashCmd("cd " + this.Application.SrcDir() + " && git checkout -q -f " + this.Revision)
	if this.err != nil {
		return this.err
	}
	// Convert references to submodules to be read-only.
	this.err = e.BashCmd(`test -f '` + this.Application.SrcDir() + `/.gitmodules' && echo 'git: converting submodule refs to be read-only' && sed -i 's,git@github.com:,git://github.com/,g' '` + this.Application.SrcDir() + `/.gitmodules' || echo 'git: project does not appear to have any submodules'`)
	if this.err != nil {
		return this.err
	}
	// Update the submodules.
	this.err = e.BashCmd("cd " + this.Application.SrcDir() + " && git submodule init && git submodule update")
	if this.err != nil {
		return this.err
	}
	// Clear out and remove all git files from the container; they are unnecessary from this point forward.
	// NB: If this command fails, don't abort anything, just log the error.
	ignorableErr := e.BashCmd(`find ` + this.Application.SrcDir() + ` . -regex '^.*\.git\(ignore\|modules\|attributes\)?$' -exec rm -rf {} \; 1>/dev/null 2>/dev/null`)
	if ignorableErr != nil {
		fmt.Fprintf(dimLogger, ".git* cleanup failed: %v\n", ignorableErr)
	}
	return nil
}

func (this *Deployment) prepareEnvironmentVariables(e *Executor) error {
	// Write out the environmental variables.
	err := e.BashCmd("rm -rf " + this.Application.AppDir() + "/env")
	if err != nil {
		return err
	}
	err = e.BashCmd("mkdir -p " + this.Application.AppDir() + "/env")
	if err != nil {
		return err
	}
	for key, value := range this.Application.Environment {
		err = ioutil.WriteFile(this.Application.AppDir()+"/env/"+key, []byte(value), 0444)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *Deployment) prepareShellEnvironment(e *Executor) error {
	// Update the container's /etc/passwd file to use the `envdirbash` script and /app/src as the user's home directory.
	escapedAppSrc := strings.Replace(this.Application.LocalSrcDir(), "/", `\/`, -1)
	err := e.Run("sudo",
		"sed", "-i",
		`s/^\(`+DEFAULT_NODE_USERNAME+`:.*:\):\/home\/`+DEFAULT_NODE_USERNAME+`:\/bin\/bash$/\1:`+escapedAppSrc+`:\/bin\/bash/g`,
		this.Application.RootFsDir()+"/etc/passwd",
	)
	if err != nil {
		return err
	}
	// Move /home/<user>/.ssh to the new home directory in /app/src
	err = e.BashCmd("cp -a /home/" + DEFAULT_NODE_USERNAME + "/.[a-zA-Z0-9]* " + this.Application.SrcDir() + "/")
	if err != nil {
		return err
	}
	return nil
}

func (this *Deployment) prepareAppFilePermissions(e *Executor) error {
	// Chown the app src & output to default user by grepping the uid+gid from /etc/passwd in the container.
	return e.BashCmd(
		"touch " + this.Application.AppDir() + "/out && " +
			"chown $(cat " + this.Application.RootFsDir() + "/etc/passwd | grep '^" + DEFAULT_NODE_USERNAME + ":' | cut -d':' -f3,4) " +
			this.Application.AppDir() + " && " +
			"chown -R $(cat " + this.Application.RootFsDir() + "/etc/passwd | grep '^" + DEFAULT_NODE_USERNAME + ":' | cut -d':' -f3,4) " +
			this.Application.AppDir() + "/{out,src}",
	)
}

// Disable unnecessary services in container.
func (this *Deployment) prepareDisabledServices(e *Executor) error {
	// Disable `ondemand` power-saving service by unlinking it from /etc/rc*.d.
	err := e.BashCmd(`find ` + this.Application.RootFsDir() + `/etc/rc*.d/ -wholename '*/S*ondemand' -exec unlink {} \;`)
	if err != nil {
		return err
	}
	// Disable `ntpdate` client from being triggered when networking comes up.
	err = e.BashCmd(`chmod a-x ` + this.Application.RootFsDir() + `/etc/network/if-up.d/ntpdate`)
	if err != nil {
		return err
	}
	// Disable auto-start for unnecessary services in /etc/init/*.conf, such as: SSH, rsyslog, cron, tty1-6, and udev.
	for _, service := range []string{"ssh", "rsyslog", "cron", "tty1", "tty2", "tty3", "tty4", "tty5", "tty6", "udev"} {
		err = e.BashCmd("echo 'manual' > " + this.Application.RootFsDir() + "/etc/init/" + service + ".override")
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *Deployment) build() error {
	dimLogger := NewFormatter(this.Logger, DIM)
	titleLogger := NewFormatter(this.Logger, GREEN)

	e := &Executor{dimLogger}

	fmt.Fprintf(titleLogger, "Building image\n")
	e.StopContainer(this.Application.Name) // To be sure we are starting with a container in the stopped state.

	// Create upstart script.
	f, err := os.OpenFile(this.Application.RootFsDir()+"/etc/init/app.conf", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0444)
	if err != nil {
		return err
	}
	err = UPSTART.Execute(f, nil)
	f.Close()
	if err != nil {
		return err
	}
	// Create the build script.
	f, err = os.OpenFile(this.Application.RootFsDir()+APP_DIR+"/run", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	err = BUILD_PACKS[this.Application.BuildPack].Execute(f, nil)
	f.Close()
	if err != nil {
		return err
	}
	// Create a file to store container launch output in.
	f, err = os.Create(this.Application.AppDir() + "/out")
	if err != nil {
		return err
	}
	defer f.Close()

	c := make(chan error)
	go func() {
		buf := make([]byte, 8192)
		var err error
		for {
			n, _ := f.Read(buf)
			if n > 0 {
				dimLogger.Write(buf[:n])
				if bytes.Contains(buf, []byte("RETURN_CODE")) {
					if !bytes.Contains(buf, []byte("RETURN_CODE: 0")) {
						err = fmt.Errorf("build failed")
					}
					break
				}
			} else {
				time.Sleep(time.Millisecond * 100)
			}
		}
		c <- err
	}()
	err = e.StartContainer(this.Application.Name)
	if err != nil {
		return err
	}

	select {
	case err = <-c:
	case <-time.After(30 * time.Minute):
		err = fmt.Errorf("timeout")
	}
	e.StopContainer(this.Application.Name)
	if err != nil {
		return err
	}

	err = this.prepareEnvironmentVariables(e)
	if err != nil {
		return err
	}
	err = this.prepareShellEnvironment(e)
	if err != nil {
		return err
	}
	err = this.prepareAppFilePermissions(e)
	if err != nil {
		return err
	}
	err = this.prepareDisabledServices(e)
	if err != nil {
		return err
	}

	return nil
}

func (this *Deployment) archive() error {
	dimLogger := NewFormatter(this.Logger, DIM)

	e := Executor{dimLogger}

	versionedContainerName := this.Application.Name + DYNO_DELIMITER + this.Version

	e.CloneContainer(this.Application.Name, versionedContainerName)

	// Compress & upload the image to S3.
	go func() {
		e = Executor{NewLogger(os.Stdout, "[archive] ")}
		archiveName := "/tmp/" + versionedContainerName + ".tar.gz"
		err := e.BashCmd("tar --create --gzip --preserve-permissions --file " + archiveName + " " + this.Application.RootFsDir())
		if err != nil {
			return
		}
		defer e.BashCmd("rm -f " + archiveName)

		h, err := os.Open(archiveName)
		if err != nil {
			return
		}
		defer h.Close()
		stat, err := h.Stat()
		if err != nil {
			return
		}
		getS3Bucket().PutReader(
			"/releases/"+this.Application.Name+"/"+this.Version+".tar.gz",
			h,
			stat.Size(),
			"application/x-tar-gz",
			"private",
		)
	}()
	return nil
}

func (this *Deployment) extract(version string) error {
	e := Executor{this.Logger}

	err := this.Application.CreateBaseContainerIfMissing(&e)
	if err != nil {
		return err
	}

	// Detect if the container is already present locally.
	versionedAppContainer := this.Application.Name + DYNO_DELIMITER + version
	if e.ContainerExists(versionedAppContainer) {
		fmt.Fprintf(this.Logger, "Syncing local copy of %v\n", version)
		// Rsync to versioned container to base app container.
		rsyncCommand := "rsync --recursive --links --hard-links --devices --specials --acls --owner --perms --times --delete --xattrs --numeric-ids "
		return e.BashCmd(rsyncCommand + LXC_DIR + "/" + versionedAppContainer + "/rootfs/ " + this.Application.RootFsDir())
	}

	// The requested app version doesn't exist locally, attempt to download it from S3.
	return extractAppFromS3(&e, this.Application, version)
}

func extractAppFromS3(e *Executor, app *Application, version string) error {
	fmt.Fprintf(e.logger, "Downloading release %v from S3\n", version)
	r, err := getS3Bucket().GetReader("/releases/" + app.Name + "/" + version + ".tar.gz")
	if err != nil {
		return err
	}
	defer r.Close()

	localArchive := "/tmp/" + app.Name + DYNO_DELIMITER + version + ".tar.gz"
	h, err := os.Create(localArchive)
	if err != nil {
		return err
	}
	defer h.Close()
	defer os.Remove(localArchive)

	_, err = io.Copy(h, r)
	if err != nil {
		return err
	}

	fmt.Fprintf(e.logger, "Extracting %v\n", localArchive)
	err = e.BashCmd("rm -rf " + app.RootFsDir() + "/*")
	if err != nil {
		return err
	}
	err = e.BashCmd("tar -C / --extract --gzip --preserve-permissions --file " + localArchive)
	if err != nil {
		return err
	}
	return nil
}

func (this *Deployment) syncNode(node *Node) error {
	logger := NewLogger(this.Logger, "["+node.Host+"] ")
	e := Executor{logger}

	// TODO: Maybe add fail check to clone operation.
	err := e.Run("ssh", DEFAULT_NODE_USERNAME+"@"+node.Host,
		"sudo", "/bin/bash", "-c",
		`"test ! -d '`+LXC_DIR+`/`+this.Application.Name+`' && lxc-clone -B `+lxcFs+` -s -o base-`+this.Application.BuildPack+` -n `+this.Application.Name+` || echo 'app image already exists'"`,
	)
	if err != nil {
		fmt.Fprintf(logger, "error cloning base container: %v\n", err)
		return err
	}
	// Rsync the application container over.
	err = e.Run("sudo", "rsync",
		"--recursive",
		"--links",
		"--hard-links",
		"--devices",
		"--specials",
		"--owner",
		"--perms",
		"--times",
		"--acls",
		"--delete",
		"--xattrs",
		"--numeric-ids",
		"-e", "ssh "+DEFAULT_SSH_PARAMETERS,
		this.Application.LxcDir()+"/rootfs/",
		"root@"+node.Host+":"+this.Application.LxcDir()+"/rootfs/",
	)
	if err != nil {
		return err
	}
	err = e.Run("rsync",
		"-azve", "ssh "+DEFAULT_SSH_PARAMETERS,
		"/tmp/postdeploy.py", "/tmp/shutdown_container.py",
		"root@"+node.Host+":/tmp/",
	)
	if err != nil {
		return err
	}
	return nil
}

func (this *Deployment) startDyno(dynoGenerator *DynoGenerator, process string) (Dyno, error) {
	dyno := dynoGenerator.Next(process)

	logger := NewLogger(this.Logger, "["+dyno.Host+"] ")
	e := Executor{logger}

	var err error
	done := make(chan bool)
	go func() {
		fmt.Fprint(logger, "Starting dyno")
		err = e.Run("ssh", DEFAULT_NODE_USERNAME+"@"+dyno.Host, "sudo", "/tmp/postdeploy.py", dyno.Container)
		done <- true
	}()
	select {
	case <-done: // implicitly break.
	case <-time.After(DYNO_START_TIMEOUT_SECONDS * time.Second):
		err = fmt.Errorf("Timed out for dyno host %v", dyno.Host)
	}
	return dyno, err
}

func (this *Deployment) autoDetectRevision() error {
	revision, err := ioutil.ReadFile(this.Application.SrcDir() + "/.git/HEAD")
	if err != nil {
		return err
	}
	this.Revision = strings.Trim(string(revision), "\n")
	return nil
}

func writeDeployScripts() error {
	err := ioutil.WriteFile("/tmp/postdeploy.py", []byte(POSTDEPLOY), 0777)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("/tmp/shutdown_container.py", []byte(SHUTDOWN_CONTAINER), 0777)
	if err != nil {
		return err
	}
	return nil
}

func (this *Deployment) calculateDynosToDestroy() ([]Dyno, bool, error) {
	// Track whether or not new dynos will be allocated.  If no new allocations are necessary, no rsync'ing will be necessary.
	allocatingNewDynos := false
	// Build list of running dynos to be deactivated in the LB config upon successful deployment.
	removeDynos := []Dyno{}
	for process, numDynos := range this.Application.Processes {
		runningDynos, err := this.Server.GetRunningDynos(this.Application.Name, process)
		if err != nil {
			return removeDynos, allocatingNewDynos, err
		}
		if !this.ScalingOnly {
			removeDynos = append(removeDynos, runningDynos...)
			allocatingNewDynos = true
		} else if numDynos < 0 {
			// Scaling down this type of process.
			if len(runningDynos) >= -1*numDynos {
				// NB: -1*numDynos in this case == positive number of dynos to remove.
				removeDynos = append(removeDynos, runningDynos[0:-1*numDynos]...)
			} else {
				removeDynos = append(removeDynos, runningDynos...)
			}
		} else {
			allocatingNewDynos = true
		}
	}
	fmt.Fprintf(this.Logger, "calculateDynosToDestroy :: calculated to remove the following dynos: %v\n", removeDynos)
	return removeDynos, allocatingNewDynos, nil
}

func (this *Deployment) syncNodes() ([]*Node, error) {
	type NodeSyncResult struct {
		node *Node
		err  error
	}

	syncStep := make(chan NodeSyncResult)
	for _, node := range this.Config.Nodes {
		go func(node *Node) {
			c := make(chan error, 1)
			go func() { c <- this.syncNode(node) }()
			go func() {
				time.Sleep(NODE_SYNC_TIMEOUT_SECONDS * time.Second)
				c <- fmt.Errorf("Sync operation to node '%v' timed out after %v seconds", node.Host, NODE_SYNC_TIMEOUT_SECONDS)
			}()
			// Block until chan has something, at which point syncStep will be notified.
			syncStep <- NodeSyncResult{node, <-c}
		}(node)
	}

	availableNodes := []*Node{}

	// Wait for all the syncs to finish or timeout, and collect available nodes.
	for _ = range this.Config.Nodes {
		syncResult := <-syncStep
		if syncResult.err == nil {
			availableNodes = append(availableNodes, syncResult.node)
		}
	}

	if len(availableNodes) == 0 {
		return availableNodes, fmt.Errorf("No available nodes. This is probably very bad for all apps running on this deployment system.")
	}
	return availableNodes, nil
}

func (this *Deployment) startDynos(availableNodes []*Node, titleLogger io.Writer) ([]Dyno, error) {
	// Now we've successfully sync'd and we have a list of nodes available to deploy to.
	addDynos := []Dyno{}

	dynoGenerator, err := this.Server.NewDynoGenerator(availableNodes, this.Application.Name, this.Version)
	if err != nil {
		return addDynos, err
	}

	type StartResult struct {
		dyno Dyno
		err  error
	}
	startedChannel := make(chan StartResult)

	startDynoWrapper := func(dynoGenerator *DynoGenerator, process string) {
		dyno, err := this.startDyno(dynoGenerator, process)
		startedChannel <- StartResult{dyno, err}
	}

	numDesiredDynos := 0

	// First deploy the changes and start the new dynos.
	for process, numDynos := range this.Application.Processes {
		for i := 0; i < numDynos; i++ {
			go startDynoWrapper(dynoGenerator, process)
			numDesiredDynos++
		}
	}

	if numDesiredDynos > 0 {
		timeout := time.After(DEPLOY_TIMEOUT_SECONDS * time.Second)
	OUTER:
		for {
			select {
			case result := <-startedChannel:
				if result.err != nil {
					// Then attempt start it again.
					fmt.Fprintf(titleLogger, "Retrying starting app dyno %v on host %v, failure reason: %v\n", result.dyno.Process, result.dyno.Host, result.err)
					go startDynoWrapper(dynoGenerator, result.dyno.Process)
				} else {
					addDynos = append(addDynos, result.dyno)
					if len(addDynos) == numDesiredDynos {
						fmt.Fprintf(titleLogger, "Successfully started app on %v total dynos\n", numDesiredDynos)
						break OUTER
					}
				}
			case <-timeout:
				return addDynos, fmt.Errorf("Start operation timed out after %v seconds", DEPLOY_TIMEOUT_SECONDS)
			}
		}
	}
	return addDynos, nil
}

// Deploy and launch the container to nodes.
func (this *Deployment) deploy() error {
	if len(this.Application.Processes) == 0 {
		return fmt.Errorf("No processes scaled up, adjust with `ps:scale procType=#` before deploying")
	}

	titleLogger := NewFormatter(this.Logger, GREEN)
	dimLogger := NewFormatter(this.Logger, DIM)

	e := Executor{dimLogger}

	this.autoDetectRevision()

	err := writeDeployScripts()
	if err != nil {
		return err
	}

	removeDynos, allocatingNewDynos, err := this.calculateDynosToDestroy()
	if err != nil {
		return err
	}

	if allocatingNewDynos {
		availableNodes, err := this.syncNodes()
		if err != nil {
			return err
		}

		// Now we've successfully sync'd and we have a list of nodes available to deploy to.
		addDynos, err := this.startDynos(availableNodes, titleLogger)
		if err != nil {
			return err
		}

		fmt.Fprintf(titleLogger, "Arbitrary sleeping for 30s to allow dynos to warm up before syncing load balancers\n")
		time.Sleep(30 * time.Second)

		err = this.Server.SyncLoadBalancers(&e, addDynos, removeDynos)
		if err != nil {
			return err
		}
	}

	if !this.ScalingOnly {
		// Update releases.
		releases, err := getReleases(this.Application.Name)
		if err != nil {
			return err
		}
		// Prepend the release (releases are in descending order)
		releases = append([]Release{{
			Version:  this.Version,
			Revision: this.Revision,
			Date:     time.Now(),
			Config:   this.Application.Environment,
		}}, releases...)
		// Only keep around the latest 15 (older ones are still in S3)
		if len(releases) > 15 {
			releases = releases[:15]
		}
		err = setReleases(this.Application.Name, releases)
		if err != nil {
			return err
		}
	} else {
		// Trigger old dynos to shutdown.
		for _, removeDyno := range removeDynos {
			fmt.Fprintf(titleLogger, "Shutting down dyno: %v\n", removeDyno.Container)
			go func(rd Dyno) {
				rd.Shutdown(&Executor{os.Stdout})
			}(removeDyno)
		}
	}

	return nil
}

func (this *Deployment) postDeployHooks(err error) {
	var message string
	notify := "0"
	color := "green"

	revision := "."
	if len(this.Revision) > 0 {
		revision = " (" + this.Revision[0:7] + ")."
	}

	durationFractionStripper, _ := regexp.Compile(`^(.*)\.[0-9]*(s)?$`)
	duration := durationFractionStripper.ReplaceAllString(time.Since(this.StartedTs).String(), "$1$2")

	hookUrl, ok := this.Application.Environment["DEPLOYHOOKS_HTTP_URL"]
	if !ok {
		fmt.Printf("app '%v' doesn't have a DEPLOYHOOKS_HTTP_URL\n", this.Application.Name)
		return
	} else if err != nil {
		task := "Deployment"
		if this.ScalingOnly {
			task = "Scaling"
		}
		message = this.Application.Name + ": " + task + " operation failed after " + duration + ": " + err.Error() + revision
		notify = "1"
		color = "red"
	} else if err == nil && this.ScalingOnly {
		procInfo := ""
		err := this.Server.WithApplication(this.Application.Name, func(app *Application, cfg *Config) error {
			for proc, val := range app.Processes {
				procInfo += " " + proc + "=" + strconv.Itoa(val)
			}
			return nil
		})
		if err != nil {
			fmt.Printf("warn: postDeployHooks scaling caught: %v", err)
		}
		message = "Scaled " + this.Application.Name + " to" + procInfo + " in " + duration + revision
	} else {
		message = "Deployed " + this.Application.Name + " " + this.Version + " in " + duration + revision
	}

	if strings.HasPrefix(hookUrl, "https://api.hipchat.com/v1/rooms/message") {
		hookUrl += "&notify=" + notify + "&color=" + color + "&from=ShipBuilder&message_format=text&message=" + url.QueryEscape(message)
		fmt.Printf("info: dispatching app deployhook url, app=%v url=%v\n", this.Application.Name, hookUrl)
		go http.Get(hookUrl)
	} else {
		fmt.Printf("error: unrecognized app deployhook url, app=%v url=%v\n", this.Application.Name, hookUrl)
	}
}

func (this *Deployment) undoVersionBump() {
	e := Executor{this.Logger}
	e.DestroyContainer(this.Application.Name + DYNO_DELIMITER + this.Version)
	this.Server.WithPersistentApplication(this.Application.Name, func(app *Application, cfg *Config) error {
		// If the version hasn't been messed with since we incremented it, go ahead and decrement it because
		// this deploy has failed.
		if app.LastDeploy == this.Version {
			prev, err := app.CalcPreviousVersion()
			if err != nil {
				return err
			}
			app.LastDeploy = prev
		}
		return nil
	})
}

func (this *Deployment) Deploy() error {
	var err error

	// Cleanup any hanging chads upon error.
	defer func() {
		if err != nil {
			this.undoVersionBump()
		}
		this.postDeployHooks(err)
	}()

	if !this.ScalingOnly {
		err = this.createContainer()
		if err != nil {
			return err
		}

		err = this.build()
		if err != nil {
			return err
		}

		err = this.archive()
		if err != nil {
			return err
		}
	}

	err = this.deploy()
	if err != nil {
		return err
	}

	return nil
}

func (this *Server) Deploy(conn net.Conn, applicationName, revision string) error {
	deployLock.start()
	defer deployLock.finish()

	logger := NewTimeLogger(NewMessageLogger(conn))
	fmt.Fprintf(logger, "Deploying revision %v\n", revision)

	return this.WithApplication(applicationName, func(app *Application, cfg *Config) error {
		// Bump version.
		app, cfg, err := this.IncrementAppVersion(app)
		if err != nil {
			return err
		}
		deployment := &Deployment{
			Server:      this,
			Logger:      logger,
			Config:      cfg,
			Application: app,
			Revision:    revision,
			Version:     app.LastDeploy,
			StartedTs:   time.Now(),
		}
		err = deployment.Deploy()
		if err != nil {
			return err
		}
		return nil
	})
}

func (this *Server) Redeploy(conn net.Conn, applicationName string) error {
	deployLock.start()
	defer deployLock.finish()

	logger := NewTimeLogger(NewMessageLogger(conn))

	return this.WithApplication(applicationName, func(app *Application, cfg *Config) error {
		if app.LastDeploy == "" {
			// Nothing to redeploy.
			return fmt.Errorf("Redeploy is not going to happen because this app has not yet had a first deploy")
		}
		previousVersion := app.LastDeploy
		// Bump version.
		app, cfg, err := this.IncrementAppVersion(app)
		if err != nil {
			return err
		}
		deployment := &Deployment{
			Server:      this,
			Logger:      logger,
			Config:      cfg,
			Application: app,
			Version:     app.LastDeploy,
			StartedTs:   time.Now(),
		}
		// Find the release that corresponds with the latest deploy.
		releases, err := getReleases(applicationName)
		if err != nil {
			return err
		}
		found := false
		for _, r := range releases {
			if r.Version == previousVersion {
				deployment.Revision = r.Revision
				found = true
				break
			}
		}
		if !found {
			// Roll back the version bump.
			err = this.WithPersistentApplication(applicationName, func(app *Application, cfg *Config) error {
				app.LastDeploy = previousVersion
				return nil
			})
			if err != nil {
				return err
			}
			return fmt.Errorf("failed to find previous deploy: %v", previousVersion)
		}
		Logf(conn, "redeploying\n")
		return deployment.Deploy()
	})
}

func (this *Server) Rescale(conn net.Conn, applicationName string, args map[string]string) error {
	deployLock.start()
	defer deployLock.finish()

	logger := NewLogger(NewTimeLogger(NewMessageLogger(conn)), "[scale] ")

	// Calculate scale changes to make.
	changes := map[string]int{}

	err := this.WithPersistentApplication(applicationName, func(app *Application, cfg *Config) error {
		for processType, newNumDynosStr := range args {
			newNumDynos, err := strconv.Atoi(newNumDynosStr)
			if err != nil {
				return err
			}

			oldNumDynos, ok := app.Processes[processType]
			if !ok {
				// Add new dyno type to changes.
				changes[processType] = newNumDynos
			} else if newNumDynos != oldNumDynos {
				// Take note of difference.
				changes[processType] = newNumDynos - oldNumDynos
			}

			app.Processes[processType] = newNumDynos
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		return fmt.Errorf("No scaling changes were detected")
	}

	// Apply the changes.
	return this.WithApplication(applicationName, func(app *Application, cfg *Config) error {
		if app.LastDeploy == "" {
			// Nothing to redeploy.
			return fmt.Errorf("Rescaling will apply only to future deployments because this app has not yet had a first deploy")
		}

		fmt.Fprintf(logger, "will make the following scale adjustments: %v\n", changes)

		// Temporarily replace Processes with the diff.
		app.Processes = changes
		deployment := &Deployment{
			Server:      this,
			Logger:      logger,
			Config:      cfg,
			Application: app,
			Version:     app.LastDeploy,
			StartedTs:   time.Now(),
			ScalingOnly: true,
		}
		return deployment.Deploy()
	})
}
