package hm

import (
	"errors"
	"fmt"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/storeadapter"
	"os"
	"time"
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

	lostLockChannel, releaseLockChannel, err := adapter.GetAndMaintainLock(component, 10)
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to acquire lock: %s", err))
		return err
	}

	go func() {
		<-lostLockChannel
		logger.Error("Lost the lock", errors.New("Lock the lock"))
		os.Exit(197)
	}()

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
			releaseLockChannel <- true
			return errors.New("Daemon timed out. Aborting!")
		}

		<-afterChan
	}

	return nil
}
