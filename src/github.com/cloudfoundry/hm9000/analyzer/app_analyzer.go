package analyzer

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/models"
	"strconv"
	"time"
)

type appAnalyzer struct {
	app                          *models.App
	conf                         config.Config
	existingPendingStartMessages map[string]models.PendingStartMessage
	existingPendingStopMessages  map[string]models.PendingStopMessage
	currentTime                  time.Time
	logger                       logger.Logger

	startMessages []models.PendingStartMessage
	stopMessages  []models.PendingStopMessage
	crashCounts   []models.CrashCount
}

func newAppAnalyzer(app *models.App, currentTime time.Time, existingPendingStartMessages map[string]models.PendingStartMessage, existingPendingStopMessages map[string]models.PendingStopMessage, logger logger.Logger, conf config.Config) *appAnalyzer {
	return &appAnalyzer{
		app:  app,
		conf: conf,
		existingPendingStartMessages: existingPendingStartMessages,
		existingPendingStopMessages:  existingPendingStopMessages,
		currentTime:                  currentTime,
		logger:                       logger,
		startMessages:                make([]models.PendingStartMessage, 0),
		stopMessages:                 make([]models.PendingStopMessage, 0),
		crashCounts:                  make([]models.CrashCount, 0),
	}
}

func (a *appAnalyzer) analyzeApp() ([]models.PendingStartMessage, []models.PendingStopMessage, []models.CrashCount) {
	priority := a.computePendingStartMessagePriority()
	a.generatePendingStartsForMissingInstances(priority)
	a.generatePendingStartsForCrashedInstances(priority)

	if len(a.startMessages) == 0 {
		a.generatePendingStopsForExtraInstances()
		a.generatePendingStopsForDuplicateInstances()
	}

	return a.startMessages, a.stopMessages, a.crashCounts
}

func (a *appAnalyzer) generatePendingStartsForMissingInstances(priority float64) {
	for index := 0; a.app.IsIndexDesired(index); index++ {
		if !a.app.HasStartingOrRunningInstanceAtIndex(index) && !a.app.HasCrashedInstanceAtIndex(index) {
			message := models.NewPendingStartMessage(a.currentTime, a.conf.GracePeriod(), 0, a.app.AppGuid, a.app.AppVersion, index, priority)

			a.logger.Info("Identified missing instance", message.LogDescription(), map[string]string{
				"Desired # of Instances": strconv.Itoa(a.app.NumberOfDesiredInstances()),
			})

			a.appendStartMessageIfNotDuplicate(message)
		}
	}

	return
}

func (a *appAnalyzer) generatePendingStartsForCrashedInstances(priority float64) (crashCounts []models.CrashCount) {
	for index := 0; a.app.IsIndexDesired(index); index++ {
		if !a.app.HasStartingOrRunningInstanceAtIndex(index) && a.app.HasCrashedInstanceAtIndex(index) {
			if index != 0 && !a.app.HasStartingOrRunningInstances() {
				continue
			}

			crashCount := a.app.CrashCountAtIndex(index, a.currentTime)
			delay := a.computeDelayForCrashCount(crashCount)
			message := models.NewPendingStartMessage(a.currentTime, delay, a.conf.GracePeriod(), a.app.AppGuid, a.app.AppVersion, index, priority)

			a.logger.Info("Identified crashed instance", message.LogDescription(), map[string]string{
				"Desired # of Instances": strconv.Itoa(a.app.NumberOfDesiredInstances()),
			})

			didAppend := a.appendStartMessageIfNotDuplicate(message)

			if didAppend {
				crashCount.CrashCount += 1
				a.crashCounts = append(a.crashCounts, crashCount)
			}
		}
	}

	return
}

func (a *appAnalyzer) generatePendingStopsForExtraInstances() {
	for _, extraInstance := range a.app.ExtraStartingOrRunningInstances() {
		message := models.NewPendingStopMessage(a.currentTime, 0, a.conf.GracePeriod(), a.app.AppGuid, a.app.AppVersion, extraInstance.InstanceGuid)

		a.appendStopMessageIfNotDuplicate(message)

		a.logger.Info("Identified extra running instance", message.LogDescription(), map[string]string{
			"InstanceIndex":          strconv.Itoa(extraInstance.InstanceIndex),
			"Desired # of Instances": strconv.Itoa(a.app.NumberOfDesiredInstances()),
		})
	}

	return
}

func (a *appAnalyzer) generatePendingStopsForDuplicateInstances() {
	//stop duplicate instances at indices < numDesired
	//this works by scheduling stops for *all* duplicate instances at increasing delays
	//the sender will process the stops one at a time and only send stops that don't put
	//the system in an invalid state
	for index := 0; a.app.IsIndexDesired(index); index++ {
		instances := a.app.StartingOrRunningInstancesAtIndex(index)
		if len(instances) > 1 {
			for i, instance := range instances {
				delay := (i + 1) * a.conf.GracePeriod()
				message := models.NewPendingStopMessage(a.currentTime, delay, a.conf.GracePeriod(), a.app.AppGuid, a.app.AppVersion, instance.InstanceGuid)

				a.appendStopMessageIfNotDuplicate(message)

				a.logger.Info("Identified duplicate running instance", message.LogDescription(), map[string]string{
					"InstanceIndex": strconv.Itoa(instance.InstanceIndex),
				})
			}
		}
	}

	return
}

func (a *appAnalyzer) appendStartMessageIfNotDuplicate(message models.PendingStartMessage) (didAppend bool) {
	_, alreadyQueued := a.existingPendingStartMessages[message.StoreKey()]
	if !alreadyQueued {
		a.logger.Info("Enqueuing Start Message", message.LogDescription())
		a.startMessages = append(a.startMessages, message)
		return true
	} else {
		a.logger.Info("Skipping Already Enqueued Start Message", message.LogDescription())
		return false
	}
}

func (a *appAnalyzer) appendStopMessageIfNotDuplicate(message models.PendingStopMessage) {
	_, alreadyQueued := a.existingPendingStopMessages[message.StoreKey()]
	if !alreadyQueued {
		a.logger.Info("Enqueuing Stop Message", message.LogDescription())
		a.stopMessages = append(a.stopMessages, message)
	} else {
		a.logger.Info("Skipping Already Enqueued Stop Message", message.LogDescription())
	}
}

func (a *appAnalyzer) computePendingStartMessagePriority() float64 {
	numberOfMissingIndices := a.app.NumberOfDesiredInstances() - a.app.NumberOfDesiredIndicesWithAStartingOrRunningInstance()

	return float64(numberOfMissingIndices) / float64(a.app.NumberOfDesiredInstances())
}

func (a *appAnalyzer) computeDelayForCrashCount(crashCount models.CrashCount) (delay int) {
	startingBackoffDelay := int(a.conf.StartingBackoffDelay().Seconds())
	maximumBackoffDelay := int(a.conf.MaximumBackoffDelay().Seconds())
	return ComputeCrashDelay(crashCount.CrashCount, a.conf.NumberOfCrashesBeforeBackoffBegins, startingBackoffDelay, maximumBackoffDelay)
}
