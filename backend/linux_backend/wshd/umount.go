package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"
)

func umountAll(prefix string) error {
	mounts, err := os.Open("/proc/mounts")
	if err != nil {
		return err
	}

	defer mounts.Close()

	mountPoints := []string{}

	for {
		var skip string
		var mountPoint string

		_, err := fmt.Fscanf(
			mounts,
			"%s %s %s %s %s %s\n",
			&skip,
			&mountPoint,
			&skip,
			&skip,
			&skip,
			&skip,
		)

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if strings.HasPrefix(mountPoint, "/mnt") {
			mountPoints = append(mountPoints, mountPoint)
		}
	}

	sort.Strings(mountPoints)

	for i := len(mountPoints); i > 0; i-- {
		mountPoint := mountPoints[i-1]

		for i := 0; i < 10; i++ {
			err := syscall.Unmount(mountPoint, 0)

			if err == nil {
				break
			}

			if err != syscall.EBUSY {
				break
			}

			time.Sleep(1 * time.Second)
		}

		if err != nil {
			return err
		}
	}

	return nil
}
