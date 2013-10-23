package main

import (
	"fmt"
	"sort"
)

var help string

func init() {
	help = "Usage: " + BINARY + " COMMAND [command-specific-options] [-a APP]\n\n"

	cs := []string{}
	for _, cmd := range commands {
		cs = append(cs, cmd.LongName)
	}
	sort.Strings(cs)
	help += "Commands: \n"
	for _, cmd := range cs {
		help += "\t" + cmd + "\n"
	}
}

func (this *Local) Help(command string) {
	if command == "" {
		fmt.Print(help)
		return
	}
	for _, cmd := range commands {
		if cmd.ShortName == command || cmd.LongName == command {
			str := cmd.LongName + " "
			for _, p := range cmd.Parameters {
				if p.Type != Required {
					str += "["
				}
				str += "--" + p.Name + "=value"
				if p.Type != Required {
					str += "]"
				}
				str += " "
			}
			fmt.Println(str)
			return
		}
	}
}
