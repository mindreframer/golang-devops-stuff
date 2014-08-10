package ostent
import (
	"os"
	"fmt"
	"os/exec"
	"strings"
)

func init() {
	/*
sw_vers
sw_vers -productVersion
system_profiler SPSoftwareDataType
defaults read loginwindow SystemVersionStampAsString
defaults read /System/Library/CoreServices/SystemVersion ProductVersion
     */
	std, err := exec.Command("sw_vers", "-productVersion").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sw_vers: %s\n", err)
		return
	}
	DISTRIB = "Mac OS X " + strings.TrimRight(string(std), "\n\t ")
}
var DISTRIB string
