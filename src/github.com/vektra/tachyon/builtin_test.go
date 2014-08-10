package tachyon

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func inTmp(blk func()) {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	tmpDir := filepath.Join("test", fmt.Sprintf("builtin-test-%d", os.Getpid()))
	os.Mkdir(tmpDir, 0755)
	os.Chdir(tmpDir)

	defer os.RemoveAll(tmpDir)
	defer os.Chdir(dir)

	blk()
}

var testData = []byte("test")
var testData2 = []byte("foobar")

func TestCopySimple(t *testing.T) {
	inTmp(func() {
		ioutil.WriteFile("a.txt", testData, 0644)

		res, err := RunAdhocTask("copy", "src=a.txt dest=b.txt")
		if err != nil {
			panic(err)
		}

		if !res.Changed {
			t.Errorf("The copy didn't change anything")
		}

		data, err := ioutil.ReadFile("b.txt")
		if err != nil {
			panic(err)
		}

		if !bytes.Equal(testData, data) {
			t.Errorf("The copy didn't move the righte bytes")
		}
	})
}

func TestCopyFailsOnMissingSrc(t *testing.T) {
	inTmp(func() {
		_, err := RunAdhocTask("copy", "src=a.txt dest=b.txt")
		if err == nil {
			t.Errorf("Copy did not fail")
		}
	})
}

func TestCopyShowsNoChangeWhenFilesTheSame(t *testing.T) {
	inTmp(func() {
		ioutil.WriteFile("a.txt", testData, 0644)
		ioutil.WriteFile("b.txt", testData, 0644)

		res, err := RunAdhocTask("copy", "src=a.txt dest=b.txt")
		if err != nil {
			panic(err)
		}

		if res.Changed {
			t.Errorf("The copy changed something incorrectly")
		}

		if res.Data["md5sum"].Read().(string) == "" {
			t.Errorf("md5sum not returned")
		}
	})
}

func TestCopyMakesFileInDir(t *testing.T) {
	inTmp(func() {
		ioutil.WriteFile("a.txt", testData, 0644)
		os.Mkdir("b", 0755)

		res, err := RunAdhocTask("copy", "src=a.txt dest=b")
		if err != nil {
			panic(err)
		}

		if !res.Changed {
			t.Errorf("The copy didn't change anything")
		}

		data, err := ioutil.ReadFile("b/a.txt")
		if err != nil {
			panic(err)
		}

		if !bytes.Equal(testData, data) {
			t.Errorf("The copy didn't move the righte bytes")
		}
	})
}

func TestCopyRemovesALink(t *testing.T) {
	inTmp(func() {
		ioutil.WriteFile("a.txt", testData, 0644)
		ioutil.WriteFile("c.txt", testData2, 0644)

		os.Symlink("c.txt", "b.txt")

		res, err := RunAdhocTask("copy", "src=a.txt dest=b.txt")
		if err != nil {
			panic(err)
		}

		if !res.Changed {
			t.Errorf("The copy didn't change anything")
		}

		stat, err := os.Stat("b.txt")

		if !stat.Mode().IsRegular() {
			t.Errorf("copy didn't remove the link")
		}

		data, err := ioutil.ReadFile("b.txt")
		if err != nil {
			panic(err)
		}

		if !bytes.Equal(testData, data) {
			t.Errorf("The copy didn't move the righte bytes")
		}

		data, err = ioutil.ReadFile("c.txt")
		if err != nil {
			panic(err)
		}

		if !bytes.Equal(testData2, data) {
			t.Errorf("c.txt was overriden improperly")
		}
	})
}

func TestCopyPreservesMode(t *testing.T) {
	inTmp(func() {
		ioutil.WriteFile("a.txt", testData, 0755)

		_, err := RunAdhocTask("copy", "src=a.txt dest=b.txt")
		if err != nil {
			panic(err)
		}

		stat, err := os.Stat("b.txt")
		if err != nil {
			panic(err)
		}

		if stat.Mode().Perm() != 0755 {
			t.Errorf("Copy didn't preserve the perms")
		}
	})
}

func TestCommand(t *testing.T) {
	res, err := RunAdhocTask("command", "date")
	if err != nil {
		panic(err)
	}

	if !res.Changed {
		t.Errorf("changed not properly set")
	}

	if res.Data["rc"].Read().(int) != 0 {
		t.Errorf("return code not captured")
	}

	if res.Data["stdout"].Read().(string) == "" {
		t.Errorf("stdout was not captured: '%s'", res.Data["stdout"].Read())
	}
}

func TestShell(t *testing.T) {
	res, err := RunAdhocTask("shell", "echo \"hello dear\"")
	if err != nil {
		panic(err)
	}

	if res.Data["rc"].Read().(int) != 0 {
		t.Errorf("return code not captured")
	}

	if res.Data["stdout"].Read().(string) != "hello dear" {
		t.Errorf("stdout was not captured: '%s'", res.Data["stdout"].Read())
	}
}

func TestShellSeesNonZeroRC(t *testing.T) {
	res, err := RunAdhocTask("shell", "exit 1")
	if err != nil {
		panic(err)
	}

	if res.Data["rc"].Read().(int) != 1 {
		t.Errorf("return code not captured")
	}

	if res.Data["stdout"].Read().(string) != "" {
		t.Errorf("stdout was not captured")
	}
}

func TestScriptExecutesRelative(t *testing.T) {
	res, err := RunAdhocTask("script", "test/test_script.sh")
	if err != nil {
		panic(err)
	}

	if res.Data["rc"].Read().(int) != 0 {
		t.Errorf("return code not captured")
	}

	if res.Data["stdout"].Read().(string) != "hello script" {
		t.Errorf("stdout was not captured")
	}
}
