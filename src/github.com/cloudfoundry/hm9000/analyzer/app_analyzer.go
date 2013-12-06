package analyzer

import (
	"fmt"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/models"
	"strconv"
	"time"
)

type appAnalyzer struct {
	app                          *models.App
	conf                         *config.Config
	existingPendingStartMessages map[string]models.PendingStartMessage
	existingPendingStopMessages  map[string]models.PendingStopMessage
	currentTime                  time.Time
	logger                       logger.Logger

	startMessages map[string]models.PendingStartMessage
	stopMessages  map[string]models.PendingStopMessage
	crashCounts   []models.CrashCount
}

func newAppAnalyzer(app *models.App, currentTime time.Time, existingPendingStartMessages map[string]models.PendingStartMessage, existingPendingStopMessages map[string]models.PendingStopMessage, logger logger.Logger, conf *config.Config) *appAnalyzer {
	return &appAnalyzer{
		app:  app,
		conf: conf,
		existingPendingStartMessages: existingPendingStartMessages,
		existingPendingStopMessages:  existingPendingStopMessages,
		currentTime:                  currentTime,
		logger:                       logger,
		startMessages:                make(map[string]models.PendingStartMessage, 0),
		stopMessages:                 make(map[string]models.PendingStopMessage, 0),
		crashCounts:                  make([]models.CrashCount, 0),
	}
}

func (a *appAnalyzer) analyzeApp() (map[string]models.PendingStartMessage, map[string]models.PendingStopMessage, []models.CrashCount) {
	priority := a.computePendingStartMessagePriority()
	a.generatePendingStartsForMissingInstances(priority)
	a.generatePendingStartsForCrashedInstances(priority)
	a.generatePendingStartsAndStopsForEvacuatingInstances()

	if len(a.startMessages) == 0 {
		a.generatePendingStopsForExtraInstances()
		a.generatePendingStopsForDuplicateInstances()
	}

	return a.startMessages, a.stopMessages, a.crashCounts
}

func (a *appAnalyzer) generatePendingStartsForMissingInstances(priority float64) {
	if !a.app.IsStaged() {
		return
	}

	for index := 0; a.app.IsIndexDesired(index); index++ {
		if !a.app.HasStartingOrRunningInstanceAtIndex(index) && !a.app.HasCrashedInstanceAtIndex(index) {
			message := models.NewPendingStartMessage(a.currentTime, a.conf.GracePeriod(), 0, a.app.AppGuid, a.app.AppVersion, index, priority, models.PendingStartMessageReasonMissing)

			a.appendStartMessageIfNotDuplicate(message, "Identified missing instance", map[string]string{
				"Desired # of Instances": strconv.Itoa(a.app.NumberOfDesiredInstances()),
			})
		}
	}

	return
}

func (a *appAnalyzer) generatePendingStartsForCrashedInstances(priority float64) (crashCounts []models.CrashCount) {
	if !a.app.IsStaged() {
		return
	}

	for index := 0; a.app.IsIndexDesired(index); index++ {
		if !a.app.HasStartingOrRunningInstanceAtIndex(index) && a.app.HasCrashedInstanceAtIndex(index) {
			if index != 0 && !a.app.HasStartingOrRunningInstances() {
				continue
			}

			crashCount := a.app.CrashCountAtIndex(index, a.currentTime)
			delay := a.computeDelayForCrashCount(crashCount)
			message := models.NewPendingStartMessage(a.currentTime, delay, a.conf.GracePeriod(), a.app.AppGuid, a.app.AppVersion, index, priority, models.PendingStartMessageReasonCrashed)

			didAppend := a.appendStartMessageIfNotDuplicate(message, "Identified crashed instance", map[string]string{
				"Desired # of Instances": strconv.Itoa(a.app.NumberOfDesiredInstances()),
				"Crash Count":            strconv.Itoa(crashCount.CrashCount),
			})

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
		message := models.NewPendingStopMessage(a.currentTime, 0, a.conf.GracePeriod(), a.app.AppGuid, a.app.AppVersion, extraInstance.InstanceGuid, models.PendingStopMessageReasonExtra)

		a.appendStopMessageIfNotDuplicate(message, "Identified extra running instance", map[string]string{
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
			minimumDuplicateInstanceStopDelay := 4 * a.conf.GracePeriod()

			for i, instance := range instances {
				delay := i*a.conf.GracePeriod() + minimumDuplicateInstanceStopDelay
				message := models.NewPendingStopMessage(a.currentTime, delay, a.conf.GracePeriod(), a.app.AppGuid, a.app.AppVersion, instance.InstanceGuid, models.PendingStopMessageReasonDuplicate)

				a.appendStopMessageIfNotDuplicate(message, "Identified duplicate running instance", map[string]string{
					"InstanceIndex": strconv.Itoa(instance.InstanceIndex),
				})
			}
		}
	}

	return
}

func (a *appAnalyzer) generatePendingStartsAndStopsForEvacuatingInstances() {
	heartbeatsByIndex := a.app.HeartbeatsByIndex()

	for index := range heartbeatsByIndex {
		evacuatingInstances := a.app.EvacuatingInstancesAtIndex(index)

		if len(evacuatingInstances) > 0 {
			startMessage := models.NewPendingStartMessage(a.currentTime, 0, a.conf.GracePeriod(), a.app.AppGuid, a.app.AppVersion, index, 2.0, models.PendingStartMessageReasonEvacuating)
			addStopMessages := func(displayReason string, stopReason models.PendingStopMessageReason) {
				for _, evacuatingInstance := range evacuatingInstances {
					stopMessage := models.NewPendingStopMessage(a.currentTime, 0, a.conf.GracePeriod(), a.app.AppGuid, a.app.AppVersion, evacuatingInstance.InstanceGuid, stopReason)
					a.appendStopMessageIfNotDuplicate(stopMessage, displayReason, map[string]string{})
				}
			}

			if !a.app.IsIndexDesired(index) {
				addStopMessages("Identified undesired evacuating instance.", models.PendingStopMessageReasonExtra)
				continue
			}

			if !a.app.IsStaged() {
				addStopMessages("Identified evacuating instance that is not staged.", models.PendingStopMessageReasonEvacuationComplete)
			}

			if a.app.HasRunningInstanceAtIndex(index) {
				addStopMessages("Stopping an evacuating instance that has started running elsewhere.", models.PendingStopMessageReasonEvacuationComplete)
				continue
			}

			if a.app.HasStartingInstanceAtIndex(index) {
				continue
			}

			a.appendStartMessageIfNotDuplicate(startMessage, "An instance is evacuating.  Starting it elsewhere.", map[string]string{})

			if a.app.CrashCountAtIndex(index, a.currentTime).CrashCount >= a.conf.NumberOfCrashesBeforeBackoffBegins {
				addStopMessages("Stopping an unstable evacuating instance.", models.PendingStopMessageReasonEvacuationComplete)
			}
		}
	}
}

func (a *appAnalyzer) appendStartMessageIfNotDuplicate(message models.PendingStartMessage, loggingMessage string, additionalDetails map[string]string) (didAppend bool) {
	existingMessage, alreadyQueued := a.existingPendingStartMessages[message.StoreKey()]
	if !alreadyQueued {
		a.logger.Info(fmt.Sprintf("Enqueuing Start Message: %s", loggingMessage), message.LogDescription(), additionalDetails)
		a.startMessages[message.StoreKey()] = message
		return true
	} else {
		a.logger.Info(fmt.Sprintf("Skipping Already Enqueued Start Message: %s", loggingMessage), existingMessage.LogDescription(), additionalDetails)
		return false
	}
}

func (a *appAnalyzer) appendStopMessageIfNotDuplicate(message models.PendingStopMessage, loggingMessage string, additionalDetails map[string]string) {
	existingMessage, alreadyQueued := a.existingPendingStopMessages[message.StoreKey()]
	if !alreadyQueued {
		a.logger.Info(fmt.Sprintf("Enqueuing Stop Message: %s", loggingMessage), message.LogDescription(), additionalDetails)
		a.stopMessages[message.StoreKey()] = message
	} else {
		a.logger.Info(fmt.Sprintf("Skipping Already Enqueued Stop Message: %s", loggingMessage), existingMessage.LogDescription(), additionalDetails)
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
