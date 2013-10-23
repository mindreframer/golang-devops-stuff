package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func (this *Server) numDynosAtVersion(applicationName, version string, hostStatusMap *map[string]NodeStatus) (int, error) {
	numFound := 0
	for _, nodeStatus := range *hostStatusMap {
		dynos, err := NodeStatusToDynos(&nodeStatus)
		if err != nil {
			return numFound, err
		}
		for _, dyno := range dynos {
			if dyno.Application == applicationName && dyno.Version == version {
				numFound += 1
			}
		}
	}
	return numFound, nil
}

func (this *Server) pruneDynos(nodeStatus NodeStatus, hostStatusMap *map[string]NodeStatus) error {
	cfg, err := this.getConfig(true)
	if err != nil {
		return err
	}

	logger := NewLogger(os.Stdout, "[dyno-cleanup] ")
	//fmt.Fprint(logger, "Pruning inactive dynos\n")

	type Key struct {
		application string
		version     string
		process     string
	}

	appMap := map[Key]bool{}

	appsByName := map[string]*Application{}

	// Build mapping of current expected app-process-versions.
	for _, app := range cfg.Applications {
		appsByName[app.Name] = app
		for process, _ := range app.Processes {
			//fmt.Fprintf(logger, "Existing app found, name=%v version=%v\n", app.Name, app.LastDeploy)
			appMap[Key{app.Name, app.LastDeploy, process}] = true
		}
	}

	e := &Executor{logger}

	dynos, err := NodeStatusToDynos(&nodeStatus)
	if err != nil {
		return err
	}
	for _, dyno := range dynos {
		destroy := false
		key := Key{dyno.Application, dyno.Version, dyno.Process}
		_, ok := appMap[key]
		if !ok {
			// Verify that the app has some dynos running at the current version.
			app, ok := appsByName[dyno.Application]

			if ok {
				if app.TotalRequestedDynos() > 0 {
					numAtCurrentVersion, err := this.numDynosAtVersion(app.Name, app.LastDeploy, hostStatusMap)
					if err != nil {
						return err
					}
					if dyno.Version != app.LastDeploy && numAtCurrentVersion > 0 {
						fmt.Fprintf(logger, "app container '%v' looks like an old version, terminating it (%v dynos running at latest version=%v)\n", dyno.Container, numAtCurrentVersion, app.LastDeploy)
						destroy = true
					} else {
						fmt.Fprintf(logger, "app for container '%v' doesn't appear to have any dynos running at latest version=%v, refusing to take any action\n", dyno.Container, app.LastDeploy)
					}
				} else {
					fmt.Fprintf(logger, "app '%v' has no processes scaled up, terminating it\n", dyno.Application)
					destroy = true
				}
			} else {
				fmt.Fprintf(logger, "warning: unrecognized application, ignoring..'%v'\n", dyno.Application)
			}

			if destroy {
				dynoInUseByLoadBalancer, err := this.dynoRoutingActive(&dyno)
				if err != nil {
					return err
				}
				if dynoInUseByLoadBalancer {
					fmt.Fprintf(logger, "app container '%v' is still in use by the current load-balancer configuration, termination cancelled\n", dyno.Container)
					destroy = false
				}
			}

			if destroy {
				// TODO: Add LB config check to ensure that dyno.Node + "-" + dyno.Port does not appear anywhere in the haproxy config.
				//"ssh", DEFAULT_NODE_USERNAME + "@" dyno.Host,
				fmt.Fprintf(logger, "Cleaning up trash name=%v version=%v\n", dyno.Application, dyno.Version)
				go func(dyno Dyno) {
					dyno.Shutdown(e)
				}(dyno)
			}
		}
	}

	return nil
}

// Determine if a Dyno has active routes defined in the current load-balancer configuration.
func (this *Server) dynoRoutingActive(dyno *Dyno) (bool, error) {
	// LB config check to ensure that dyno.Node + "-" + dyno.Port does not appear anywhere in the haproxy config.
	// Non-web dynos have nothing to do with the load-balancer.
	if dyno.Process != "web" {
		return false, nil
	}

	config, err := this.getConfig(true)
	if err != nil {
		return true, err
	}

	// If there aren't any load-balancers, then the dyno certainly isn't being used by one.
	if len(config.LoadBalancers) == 0 {
		return false, nil
	}

	lbConfig, err := this.GetActiveLoadBalancerConfig()
	if err != nil {
		return true, err
	}

	expr := regexp.MustCompile(` backend ` + dyno.Application + ` ([^b]|b[^a]|ba[^c]|bac[^k]|back[^e]|backe[^n]|backen[^d])* ` + dyno.Host + `-` + dyno.Port)
	inUse := expr.MatchString(strings.Replace(lbConfig, "\n", " ", -1))
	return inUse, nil
}
