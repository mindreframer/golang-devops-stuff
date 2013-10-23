package main

import (
	"fmt"
	"strings"
)

type (
	ParameterType byte
	Parameter     struct {
		Name    string
		Default interface{}
		Type    ParameterType
	}
	Command struct {
		ShortName, LongName, ServerName string
		AppRead, AppWrite               bool
		Parameters                      []Parameter
	}
)

const (
	Required ParameterType = iota // iota is just like enum
	Optional
	Mapped
	List
)

var (
	commands = []Command{}
)

func (this Command) Parse(args []string) ([]interface{}, error) {
	flags := map[string]string{}
	mapped := map[string]string{}
	unassigned := []string{}

	// parse all --{name}[=]{arg} or -{short}{arg}
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			if len(args[i]) == 2 {
				return nil, fmt.Errorf("expected `--name=value` or `--name value` but got `--`")
			} else if strings.Contains(args[i], "=") {
				vs := strings.SplitN(args[i], "=", 2)
				flags[vs[0][2:]] = vs[1]
			} else if (i + 1) < len(args) {
				flags[args[i][2:]] = args[i+1]
				i++
			} else {
				return nil, fmt.Errorf("expected `%v=value` or `%v value`", args[i], args[i])
			}
		} else if strings.HasPrefix(args[i], "-") && len(args[i]) >= 1 {
			if len(args[i]) > 1 {
				flags[args[i][1:2]] = args[i][2:]
			} else {
				flags[args[i][1:2]] = "true"
			}
		} else if strings.Contains(args[i], "=") {
			pair := strings.SplitN(args[i], "=", 2)
			mapped[pair[0]] = pair[1]
		} else {
			unassigned = append(unassigned, args[i])
		}
	}

	final := make([]interface{}, len(this.Parameters))
	for i, p := range this.Parameters {
		// long form
		if v, ok := flags[p.Name]; ok {
			final[i] = v
			continue
		}

		// short form
		if v, ok := flags[p.Name[0:1]]; ok {
			final[i] = v
			// remove it so it can't be re-used
			delete(flags, p.Name[0:1])
			continue
		}

		if p.Type == Mapped {
			final[i] = mapped
			continue
		}

		if p.Type == List {
			final[i] = unassigned
			continue
		}

		if len(unassigned) > 0 {
			final[i] = unassigned[0]
			unassigned = unassigned[1:]
			continue
		}

		if p.Type != Required {
			final[i] = p.Default
			continue
		}

		return final, fmt.Errorf("expected `%v` got `%v`", p.Name, flags)
	}

	return final, nil
}

func init() {
	required := func(name string) Parameter {
		return Parameter{
			Name:    name,
			Default: "",
			Type:    Required,
		}
	}
	mapped := func(name string) Parameter {
		return Parameter{
			Name:    name,
			Default: map[string]string{},
			Type:    Mapped,
		}
	}
	list := func(name string) Parameter {
		return Parameter{
			Name:    name,
			Default: []string{},
			Type:    List,
		}
	}
	optional := func(name, def string) Parameter {
		return Parameter{
			Name:    name,
			Default: def,
			Type:    Optional,
		}
	}
	global := func(shortName, longName, serverName string, parameters ...Parameter) Command {
		return Command{
			ShortName:  shortName,
			LongName:   longName,
			ServerName: serverName,
			Parameters: parameters,
		}
	}
	reader := func(shortName, longName, serverName string, parameters ...Parameter) Command {
		return Command{
			ShortName:  shortName,
			LongName:   longName,
			ServerName: serverName,
			AppRead:    true,
			Parameters: parameters,
		}
	}
	writer := func(shortName, longName, serverName string, parameters ...Parameter) Command {
		return Command{
			ShortName:  shortName,
			LongName:   longName,
			ServerName: serverName,
			AppRead:    true,
			AppWrite:   true,
			Parameters: parameters,
		}
	}

	commands = []Command{
		global("create", "apps:create", "Apps_Create",
			required("app"), optional("buildpack", ""),
		),
		global("destroy", "apps:destroy", "Apps_Destroy",
			required("app"),
		),
		global("clone", "apps:clone", "Apps_Clone",
			required("oldApp"), required("newApp"),
		),
		global("apps", "apps:list", "Apps_List"),
		reader("config:get", "config:get", "Config_Get",
			required("app"), required("name"),
		),
		reader("config", "config:list", "Config_List",
			required("app"),
		),
		writer("config:set", "config:add", "Config_Set",
			required("app"), optional("deferred", ""), mapped("args"),
		),
		writer("config:remove", "config:unset", "Config_Remove",
			required("app"), optional("deferred", ""), list("names"),
		),
		reader("run", "console", "Console",
			required("app"), list("args"),
		),
		writer("deploy", "deploy", "Deploy",
			required("app"), required("revision"),
		),
		writer("domains:add", "domains:add", "Domains_Add",
			required("app"), list("domains"),
		),
		reader("domains", "domains:list", "Domains_List",
			required("app"),
		),
		writer("domains:remove", "domains:remove", "Domains_Remove",
			required("app"), list("domains"),
		),
		writer("drains:add", "drains:add", "Drains_Add",
			required("app"), list("addresses"),
		),
		reader("drains", "drains:list", "Drains_List",
			required("app"),
		),
		writer("drains:remove", "drains:remove", "Drains_Remove",
			required("app"), list("addresses"),
		),
		global("help", "help", "Help",
			optional("command", ""),
		),
		global("lb:add", "lb:add", "LoadBalancer_Add",
			list("addresses"),
		),
		global("lb", "lb:list", "LoadBalancer_List"),
		global("lb:remove", "lb:remove", "LoadBalancer_Remove",
			list("addresses"),
		),
		global("logger", "logger", "Logger",
			required("host"), required("app"), required("process"),
		),
		reader("logs", "logs:get", "Logs_Get",
			required("app"), optional("process", ""), optional("filter", ""),
		),
		writer("maint:off", "maintenance:off", "Maintenance_Off",
			required("app"),
		),
		writer("maint:on", "maintenance:on", "Maintenance_On",
			required("app"),
		),
		reader("maintenance:status", "maintenance:status", "Maintenance_Status",
			required("app"),
		),
		reader("maintenance:url", "maintenance:url", "Maintenance_Url",
			required("app"), optional("url", ""),
		),
		reader("nodes:add", "nodes:add", "Node_Add",
			list("addresses"),
		),
		reader("nodes", "nodes:list", "Node_List"),
		reader("nodes:remove", "nodes:remove", "Node_Remove",
			list("addresses"),
		),
		global("post-receive", "post-receive", "PostReceive",
			required("directory"), required("oldrev"), required("newrev"), required("ref"),
		),
		global("pre-receive", "pre-receive", "PreReceive",
			required("directory"), required("oldrev"), required("newrev"), required("ref"),
		),
		reader("ps", "ps:list", "Ps_List",
			required("app"),
		),
		reader("scale", "ps:scale", "Ps_Scale",
			required("app"), mapped("args"),
		),
		writer("rollback", "rollback", "Rollback",
			required("app"), optional("version", ""),
		),
		reader("releases:info", "releases:info", "Releases_Info",
			required("app"),
		),
		reader("releases", "releases:list", "Releases_List",
			required("app"),
		),
		writer("reset", "reset", "Reset_App",
			required("app"),
		),
		writer("restart", "restart", "Restart_App",
			required("app"),
		),
	}
}
