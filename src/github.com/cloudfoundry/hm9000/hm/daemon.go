package hm

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/storeadapter"
)

func Daemonize(
	component string,
	callback func() error,
	period time.Duration,
	timeout time.Duration,
	logger logger.Logger,
	adapter storeadapter.StoreAdapter,
) error {
	logger.Info("Acquiring lock for " + component)

	lock := storeadapter.StoreNode{
		Key: "/hm/locks/" + component,
		TTL: 10,
	}

	status, releaseLockChannel, err := adapter.MaintainNode(lock)
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to acquire lock: %s", err))
		return err
	}

	lockAcquired := make(chan bool)

	go func() {
		for {
			if <-status {
				if lockAcquired != nil {
					close(lockAcquired)
					lockAcquired = nil
				}
			} else {
				logger.Error("Lost the lock", errors.New("Lock the lock"))
				os.Exit(197)
			}
		}
	}()

	<-lockAcquired

	logger.Info("Acquired lock for " + component)

	logger.Info(fmt.Sprintf("Running Daemon every %d seconds with a timeout of %d", int(period.Seconds()), int(timeout.Seconds())))

	for {
		afterChan := time.After(period)
		timeoutChan := time.After(timeout)
		errorChan := make(chan error, 1)

		t := time.Now()

		go func() {
			errorChan <- callback()
		}()

		select {
		case err := <-errorChan:
			logger.Info("Daemonize Time", map[string]string{
				"Component": component,
				"Duration":  fmt.Sprintf("%.4f", time.Since(t).Seconds()),
			})
			if err != nil {
				logger.Error("Daemon returned an error. Continuining...", err)
			}
		case <-timeoutChan:
			released := make(chan bool)
			releaseLockChannel <- released
			<-released

			return errors.New("Daemon timed out. Aborting!")
		}

		<-afterChan
	}

	return nil
}
