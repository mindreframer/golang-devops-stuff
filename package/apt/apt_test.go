package apt

import (
	"fmt"
	"github.com/vektra/tachyon"
	"os/exec"
	"testing"
)

var runAptTests = false

func init() {
	c := exec.Command("which", "apt-cache")
	c.Run()
	runAptTests = c.ProcessState.Success()
}

func TestAptDryRun(t *testing.T) {
	if !runAptTests {
		return
	}

	res, err := tachyon.RunAdhocTask("apt", "pkg=acct dryrun=true")
	if err != nil {
		panic(err)
	}

	if !res.Changed {
		t.Error("No change detected")
	}

	if res.Data.Get("installed") != "" {
		t.Error("incorrectly found an installed version")
	}

	if res.Data.Get("candidate") == "" {
		t.Error("no candidate found")
	}

	if res.Data.Get("dryrun") != true {
		t.Error("dryrun not true")
	}
}

func removeAcct() {
	exec.Command("apt-get", "remove", "-y", "--force-yes", "acct").CombinedOutput()
}

func TestAptInstallAndRemoves(t *testing.T) {
	if !runAptTests {
		return
	}

	defer removeAcct()

	res, err := tachyon.RunAdhocTask("apt", "pkg=acct")
	if err != nil {
		panic(err)
	}

	if !res.Changed {
		t.Fatal("No change detected")
	}

	grep := fmt.Sprintf(`apt-cache policy acct | grep "Installed: %s"`,
		res.Data.Get("installed"))

	_, err = exec.Command("sh", "-c", grep).CombinedOutput()

	if err != nil {
		t.Errorf("package did not install")
	}

	// Test that it skips too
	// Do this here instead of another test because installing is slow

	res2, err := tachyon.RunAdhocTask("apt", "pkg=acct")
	if err != nil {
		panic(err)
	}

	if res2.Changed {
		t.Fatal("acct was reinstalled incorrectly")
	}

	res3, err := tachyon.RunAdhocTask("apt", "pkg=acct state=absent")
	if err != nil {
		panic(err)
	}

	if !res3.Changed {
		t.Fatal("acct was not removed")
	}

	if res3.Data.Get("removed") != res.Data.Get("installed") {
		t.Fatalf("removed isn't set to the version removed: '%s '%s'",
			res3.Data.Get("removed"), res.Data.Get("installed"))
	}

	res4, err := tachyon.RunAdhocTask("apt", "pkg=acct state=absent")
	if err != nil {
		panic(err)
	}

	if res4.Changed {
		t.Fatal("acct was removed again")
	}

}
