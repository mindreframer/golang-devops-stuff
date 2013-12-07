// +build linux

package main_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"
)

var _ = Describe("Running wshd", func() {
	wshd, err := cmdtest.Build("github.com/vito/garden/backend/linux_backend/wshd")
	if err != nil {
		panic(err)
	}

	wsh, err := cmdtest.Build("github.com/vito/garden/backend/linux_backend/wshd/wsh")
	if err != nil {
		panic(err)
	}

	shmTest, err := cmdtest.Build("github.com/vito/garden/backend/linux_backend/wshd/shm_test")
	if err != nil {
		panic(err)
	}

	var socketPath string
	var containerPath string
	var wshdCommand *exec.Cmd

	BeforeEach(func() {
		containerDir, err := ioutil.TempDir(os.TempDir(), "wshd-test-container")
		Expect(err).ToNot(HaveOccured())

		containerPath = containerDir

		binDir := path.Join(containerDir, "bin")
		libDir := path.Join(containerDir, "lib")
		runDir := path.Join(containerDir, "run")
		mntDir := path.Join(containerDir, "mnt")

		os.Mkdir(binDir, 0755)
		os.Mkdir(libDir, 0755)
		os.Mkdir(runDir, 0755)

		err = copyFile(wshd, path.Join(binDir, "wshd"))
		Expect(err).ToNot(HaveOccured())

		ioutil.WriteFile(path.Join(libDir, "hook-parent-before-clone.sh"), []byte(`#!/bin/bash

set -o nounset
set -o errexit
shopt -s nullglob

cd $(dirname $0)/../

cp bin/wshd mnt/sbin/wshd
chmod 700 mnt/sbin/wshd
`), 0755)

		ioutil.WriteFile(path.Join(libDir, "hook-parent-after-clone.sh"), []byte(`#!/bin/bash

set -o nounset
set -o errexit
shopt -s nullglob

cd $(dirname $0)/../

echo $PID > ./run/wshd.pid
`), 0755)

		ioutil.WriteFile(path.Join(libDir, "hook-child-before-pivot.sh"), []byte(`#!/bin/bash
env
pwd
`), 0755)

		ioutil.WriteFile(path.Join(libDir, "hook-child-after-pivot.sh"), []byte(`#!/bin/bash

set -o nounset
set -o errexit
shopt -s nullglob

cd $(dirname $0)/../

mkdir -p /dev/pts
mount -t devpts -o newinstance,ptmxmode=0666 devpts /dev/pts
ln -sf pts/ptmx /dev/ptmx

mkdir -p /proc
mount -t proc none /proc

useradd -mU -u 10000 -s /bin/bash vcap
`), 0755)

		ioutil.WriteFile(path.Join(libDir, "set-up-root.sh"), []byte(`#!/bin/bash

set -o nounset
set -o errexit
shopt -s nullglob

rootfs_path=$1

function overlay_directory_in_rootfs() {
  # Skip if exists
  if [ ! -d tmp/rootfs/$1 ]
  then
    if [ -d mnt/$1 ]
    then
      cp -r mnt/$1 tmp/rootfs/
    else
      mkdir -p tmp/rootfs/$1
    fi
  fi

  mount -n --bind tmp/rootfs/$1 mnt/$1
  mount -n --bind -o remount,$2 tmp/rootfs/$1 mnt/$1
}

function setup_fs() {
  mkdir -p tmp/rootfs mnt

  modprobe overlayfs 2>/dev/null || true

  if grep -q -i overlayfs /proc/filesystems
  then
    mount -n -t overlayfs -o rw,upperdir=tmp/rootfs,lowerdir=$rootfs_path none mnt
  elif grep -q -i aufs /proc/filesystems
  then
    mount -n -t aufs -o br:tmp/rootfs=rw:$rootfs_path=ro+wh none mnt
  else
    mkdir -p $rootfs_path/proc

    mount -n --bind $rootfs_path mnt
    mount -n --bind -o remount,ro $rootfs_path mnt

    overlay_directory_in_rootfs /dev rw
    overlay_directory_in_rootfs /etc rw
    overlay_directory_in_rootfs /home rw
    overlay_directory_in_rootfs /sbin rw
    overlay_directory_in_rootfs /var rw

    mkdir -p tmp/rootfs/tmp
    chmod 777 tmp/rootfs/tmp
    overlay_directory_in_rootfs /tmp rw
  fi
}

setup_fs
`), 0755)

		setUpRoot := exec.Command(path.Join(libDir, "set-up-root.sh"), os.Getenv("GARDEN_TEST_ROOTFS"))
		setUpRoot.Dir = containerDir

		setUpRootSession, err := cmdtest.StartWrapped(setUpRoot, outWrapper, outWrapper)
		Expect(err).ToNot(HaveOccured())
		Expect(setUpRootSession).To(ExitWith(0))

		err = copyFile(shmTest, path.Join(mntDir, "sbin", "shmtest"))
		Expect(err).ToNot(HaveOccured())

		socketPath = path.Join(runDir, "wshd.sock")

		wshdCommand = exec.Command(
			wshd,
			"--run", runDir,
			"--lib", libDir,
			"--root", mntDir,
			"--title", "test wshd",
		)

		wshdSession, err := cmdtest.StartWrapped(wshdCommand, outWrapper, outWrapper)
		Expect(err).ToNot(HaveOccured())

		Expect(wshdSession).To(ExitWith(0))

		createdContainers = append(createdContainers, containerDir)

		Eventually(ErrorDialingUnix(socketPath)).ShouldNot(HaveOccured())
	})

	It("starts the daemon as a session leader with process isolation and the given title", func() {
		ps := exec.Command(wsh, "--socket", socketPath, "/bin/ps", "-o", "pid,command")

		psSession, err := cmdtest.StartWrapped(ps, outWrapper, outWrapper)
		Expect(err).ToNot(HaveOccured())

		Expect(psSession).To(Say(`  PID COMMAND
    1 test wshd --continue
   \d+ /bin/ps -o pid,command
`))

		Expect(psSession).ToNot(Say(`.`))

		Expect(psSession).To(ExitWith(0))
	})

	It("starts the daemon with mount space isolation", func() {
		mkdir := exec.Command(wsh, "--socket", socketPath, "/bin/mkdir", "/gnome")
		mkdirSession, err := cmdtest.StartWrapped(mkdir, outWrapper, outWrapper)
		Expect(err).ToNot(HaveOccured())
		Expect(mkdirSession).To(ExitWith(0))

		mount := exec.Command(wsh, "--socket", socketPath, "/bin/mount", "--bind", "/home", "/gnome")
		mountSession, err := cmdtest.StartWrapped(mount, outWrapper, outWrapper)
		Expect(err).ToNot(HaveOccured())
		Expect(mountSession).To(ExitWith(0))

		cat := exec.Command("/bin/cat", "/proc/mounts")
		catSession, err := cmdtest.StartWrapped(cat, outWrapper, outWrapper)
		Expect(err).ToNot(HaveOccured())
		Expect(catSession).ToNot(Say("/gnome"))
		Expect(catSession).To(ExitWith(0))
	})

	It("starts the daemon with network namespace isolation", func() {
		ifconfig := exec.Command(wsh, "--socket", socketPath, "/sbin/ifconfig", "lo:0", "1.2.3.4", "up")
		ifconfigSession, err := cmdtest.StartWrapped(ifconfig, outWrapper, outWrapper)
		Expect(err).ToNot(HaveOccured())
		Expect(ifconfigSession).To(ExitWith(0))

		localIfconfig := exec.Command("/sbin/ifconfig")
		localIfconfigSession, err := cmdtest.StartWrapped(localIfconfig, outWrapper, outWrapper)
		Expect(err).ToNot(HaveOccured())
		Expect(localIfconfigSession).ToNot(Say("lo:0"))
		Expect(localIfconfigSession).To(ExitWith(0))
	})

	It("starts the daemon with a new IPC namespace", func() {
		localSHM := exec.Command(shmTest)
		createLocal, err := cmdtest.StartWrapped(
			localSHM,
			outWrapper,
			outWrapper,
		)
		Expect(err).ToNot(HaveOccured())

		Expect(createLocal).To(Say("ok"))

		createRemote, err := cmdtest.StartWrapped(
			exec.Command(wsh, "--socket", socketPath, "/sbin/shmtest", "create"),
			outWrapper,
			outWrapper,
		)
		Expect(err).ToNot(HaveOccured())
		Expect(createRemote).To(Say("ok"))

		localSHM.Process.Signal(syscall.SIGUSR2)

		Expect(createLocal).To(ExitWith(0))
	})

	It("starts the daemon with a new UTS namespace", func() {
		hostname := exec.Command(wsh, "--socket", socketPath, "/bin/hostname", "newhostname")
		hostnameSession, err := cmdtest.StartWrapped(hostname, outWrapper, outWrapper)
		Expect(err).ToNot(HaveOccured())

		Expect(hostnameSession).To(ExitWith(0))

		localHostname := exec.Command("hostname")
		localHostnameSession, err := cmdtest.StartWrapped(localHostname, outWrapper, outWrapper)
		Expect(localHostnameSession).ToNot(Say("newhostname"))
	})

	PIt("makes the given rootfs the root of the daemon", func() {

	})

	PIt("executes the hooks in the proper sequence", func() {

	})

	PIt("does not leak file descriptors to the child", func() {
		wshdPidfile, err := os.Open(path.Join(containerPath, "run", "wshd.pid"))
		Expect(err).ToNot(HaveOccured())

		var wshdPid int

		_, err = fmt.Fscanf(wshdPidfile, "%d", &wshdPid)
		Expect(err).ToNot(HaveOccured())

		entries, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/fd", wshdPid))
		Expect(err).ToNot(HaveOccured())

		// TODO: one fd is wshd.sock, unsure what the other is.
		// shows up as type 0000 in lsof.
		//
		// compare with original wshd
		Expect(entries).To(HaveLen(2))
	})

	It("unmounts /mnt* in the child", func() {
		cat := exec.Command(wsh, "--socket", socketPath, "/bin/cat", "/proc/mounts")

		catSession, err := cmdtest.StartWrapped(cat, outWrapper, outWrapper)
		Expect(err).ToNot(HaveOccured())

		Expect(catSession).ToNot(Say(" /mnt"))
		Expect(catSession).To(ExitWith(0))
	})

	Context("when running a command as a user", func() {
		It("executes with setuid and setgid", func() {
			bash := exec.Command(wsh, "--socket", socketPath, "--user", "vcap", "/bin/bash", "-c", "id -u; id -g")

			bashSession, err := cmdtest.StartWrapped(bash, outWrapper, outWrapper)
			Expect(err).ToNot(HaveOccured())

			Expect(bashSession).To(Say("^10000\n"))
			Expect(bashSession).To(Say("^10000\n"))
			Expect(bashSession).To(ExitWith(0))
		})

		It("sets $HOME, $USER, and $PATH", func() {
			bash := exec.Command(wsh, "--socket", socketPath, "--user", "vcap", "/bin/bash", "-c", "env | sort")

			bashSession, err := cmdtest.StartWrapped(bash, outWrapper, outWrapper)
			Expect(err).ToNot(HaveOccured())

			Expect(bashSession).To(Say("HOME=/home/vcap\n"))
			Expect(bashSession).To(Say("PATH=/bin:/usr/bin\n"))
			Expect(bashSession).To(Say("USER=vcap\n"))
			Expect(bashSession).To(ExitWith(0))
		})

		It("executes in their home directory", func() {
			pwd := exec.Command(wsh, "--socket", socketPath, "--user", "vcap", "/bin/pwd")

			pwdSession, err := cmdtest.StartWrapped(pwd, outWrapper, outWrapper)
			Expect(err).ToNot(HaveOccured())

			Expect(pwdSession).To(Say("/home/vcap\n"))
			Expect(pwdSession).To(ExitWith(0))
		})
	})

	Context("when running a command as root", func() {
		It("executes with setuid and setgid", func() {
			bash := exec.Command(wsh, "--socket", socketPath, "--user", "root", "/bin/bash", "-c", "id -u; id -g")

			bashSession, err := cmdtest.StartWrapped(bash, outWrapper, outWrapper)
			Expect(err).ToNot(HaveOccured())

			Expect(bashSession).To(Say("^0\n"))
			Expect(bashSession).To(Say("^0\n"))
			Expect(bashSession).To(ExitWith(0))
		})

		It("sets $HOME, $USER, and a $PATH with sbin dirs", func() {
			bash := exec.Command(wsh, "--socket", socketPath, "--user", "root", "/bin/bash", "-c", "env | sort")

			bashSession, err := cmdtest.StartWrapped(bash, outWrapper, outWrapper)
			Expect(err).ToNot(HaveOccured())

			Expect(bashSession).To(Say("HOME=/root\n"))
			Expect(bashSession).To(Say("PATH=/sbin:/bin:/usr/sbin:/usr/bin\n"))
			Expect(bashSession).To(Say("USER=root\n"))
			Expect(bashSession).To(ExitWith(0))
		})

		It("executes in their home directory", func() {
			pwd := exec.Command(wsh, "--socket", socketPath, "--user", "root", "/bin/pwd")

			pwdSession, err := cmdtest.StartWrapped(pwd, outWrapper, outWrapper)
			Expect(err).ToNot(HaveOccured())

			Expect(pwdSession).To(Say("/root\n"))
			Expect(pwdSession).To(ExitWith(0))
		})
	})

	Context("when piping stdin", func() {
		It("terminates when the input stream terminates", func() {
			bash := exec.Command(wsh, "--socket", socketPath, "/bin/bash")

			bashSession, err := cmdtest.StartWrapped(bash, outWrapper, outWrapper)
			Expect(err).ToNot(HaveOccured())

			bashSession.Stdin.Write([]byte("echo hello"))
			bashSession.Stdin.Close()

			Expect(bashSession).To(SayWithTimeout("hello\n", 1*time.Second))
			Expect(bashSession).To(ExitWith(0))
		})
	})

	Context("when in rsh compatibility mode", func() {
		It("respects -l, discards -t [X], -46dn, skips the host, and runs the command", func() {
			pwd := exec.Command(
				wsh,
				"--socket", socketPath,
				"--user", "root",
				"--rsh",
				"-l", "vcap",
				"-t", "1",
				"-4",
				"-6",
				"-d",
				"-n",
				"somehost",
				"/bin/pwd",
			)

			pwdSession, err := cmdtest.StartWrapped(pwd, outWrapper, outWrapper)
			Expect(err).ToNot(HaveOccured())

			Expect(pwdSession).To(Say("/home/vcap\n"))
			Expect(pwdSession).To(ExitWith(0))
		})

		It("doesn't cause rsh-like flags to be consumed", func() {
			cmd := exec.Command(
				wsh,
				"--socket", socketPath,
				"--user", "root",
				"/bin/echo",
				"-l", "vcap",
				"-t", "1",
				"-4",
				"-6",
				"-d",
				"-n",
				"somehost",
			)

			cmdSession, err := cmdtest.StartWrapped(cmd, outWrapper, outWrapper)
			Expect(err).ToNot(HaveOccured())

			Expect(cmdSession).To(Say("-l vcap -t 1 -4 -6 -d -n somehost\n"))
			Expect(cmdSession).To(ExitWith(0))
		})

		It("can be used to rsync files", func() {
			cmd := exec.Command(
				"rsync",
				"-e",
				wsh+" --socket "+socketPath+" --rsh",
				"-r",
				"-p",
				"--links",
				wsh, // send wsh binary
				"root@container:wsh",
			)

			cmdSession, err := cmdtest.StartWrapped(cmd, outWrapper, outWrapper)
			Expect(err).ToNot(HaveOccured())

			Expect(cmdSession).To(ExitWith(0))
		})
	})
})

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}

	defer s.Close()

	d, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}

	_, err = io.Copy(d, s)
	if err != nil {
		d.Close()
		return err
	}

	return d.Close()
}

func outWrapper(out io.Writer) io.Writer {
	return io.MultiWriter(out, os.Stdout)
}

func ErrorDialingUnix(socketPath string) func() error {
	return func() error {
		conn, err := net.Dial("unix", socketPath)
		if err == nil {
			conn.Close()
		}

		return err
	}
}
