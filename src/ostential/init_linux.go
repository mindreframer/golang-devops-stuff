package ostential
import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func init() {
	// https://unix.stackexchange.com/q/35183
	std, err := exec.Command("lsb_release", "-i", "-r").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lsb_release: %s\n", err)
		return
	}
	id, release := "", ""
	// strings.TrimRight(string(std), "\n\t ")
	for _, s := range strings.Split(string(std), "\n") {
		b := strings.Split(s, "\t")
		if len(b) == 2 {
			if b[0] == "Distributor ID:" {
				id = b[1]
				continue
			} else if b[0] == "Release:" {
				release = b[1]
				continue
			}
		}
	}
	if id != "" && release != "" {
		DISTRIB = id + " " + release
		return
	}
	if id == "" {
		fmt.Fprintf(os.Stderr, "Could not identify Distributor ID")
	}
	if release == "" {
		fmt.Fprintf(os.Stderr, "Could not identify Release")
	}
}
