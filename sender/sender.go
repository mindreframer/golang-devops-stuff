package sender

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/yagnats"
	"sort"
)

type Sender struct {
	store  store.Store
	conf   config.Config
	logger logger.Logger

	apps         map[string]*models.App
	messageBus   yagnats.NATSClient
	timeProvider timeprovider.TimeProvider
}

func New(store store.Store, conf config.Config, messageBus yagnats.NATSClient, timeProvider timeprovider.TimeProvider, logger logger.Logger) *Sender {
	return &Sender{
		store:        store,
		conf:         conf,
		logger:       logger,
		messageBus:   messageBus,
		timeProvider: timeProvider,
	}
}

func (sender *Sender) Send() error {
	err := sender.store.VerifyFreshness(sender.timeProvider.Time())
	if err != nil {
		sender.logger.Error("Store is not fresh", err)
		return err
	}

	pendingStartMessages, err := sender.store.GetPendingStartMessages()
	if err != nil {
		sender.logger.Error("Failed to fetch pending start messages", err)
		return err
	}

	pendingStopMessages, err := sender.store.GetPendingStopMessages()
	if err != nil {
		sender.logger.Error("Failed to fetch pending stop messages", err)
		return err
	}

	sender.apps, err = sender.store.GetApps()
	if err != nil {
		sender.logger.Error("Failed to fetch apps", err)
		return err
	}

	err = sender.sendStartMessages(pendingStartMessages)
	if err != nil {
		return err
	}

	err = sender.sendStopMessages(pendingStopMessages)
	if err != nil {
		return err
	}

	return nil
}

type SortablePendingStartMessages []models.PendingStartMessage

func (s SortablePendingStartMessages) Len() int           { return len(s) }
func (s SortablePendingStartMessages) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s SortablePendingStartMessages) Less(i, j int) bool { return s[i].Priority < s[j].Priority }

func (sender *Sender) sendStartMessages(startMessages map[string]models.PendingStartMessage) error {
	startMessagesToSave := []models.PendingStartMessage{}
	startMessagesToDelete := []models.PendingStartMessage{}

	sortedStartMessages := make(SortablePendingStartMessages, len(startMessages))
	i := 0
	for _, message := range startMessages {
		sortedStartMessages[i] = message
		i++
	}
	sort.Sort(sort.Reverse(sortedStartMessages))

	numSent := 0
	maxSent := sender.conf.SenderMessageLimit

	for _, startMessage := range sortedStartMessages {
		if startMessage.IsExpired(sender.timeProvider.Time()) {
			sender.logger.Info("Deleting expired start message", startMessage.LogDescription())
			startMessagesToDelete = append(startMessagesToDelete, startMessage)
		} else if startMessage.IsTimeToSend(sender.timeProvider.Time()) {
			if sender.verifyStartMessageShouldBeSent(startMessage) {
				if numSent < maxSent {
					messageToSend := models.StartMessage{
						AppGuid:       startMessage.AppGuid,
						AppVersion:    startMessage.AppVersion,
						InstanceIndex: startMessage.IndexToStart,
						MessageId:     startMessage.MessageId,
					}
					sender.logger.Info("Sending message", startMessage.LogDescription())
					err := sender.messageBus.Publish(sender.conf.SenderNatsStartSubject, string(messageToSend.ToJSON()))
					if err != nil {
						sender.logger.Error("Failed to send start message", err, startMessage.LogDescription())
						return err
					}
					if startMessage.KeepAlive == 0 {
						sender.logger.Info("Deleting sent start message with no keep alive", startMessage.LogDescription())
						startMessagesToDelete = append(startMessagesToDelete, startMessage)
					} else {
						startMessage.SentOn = sender.timeProvider.Time().Unix()
						startMessagesToSave = append(startMessagesToSave, startMessage)
					}
					numSent += 1
				}
			} else {
				sender.logger.Info("Deleting start message that will not be sent", startMessage.LogDescription())
				startMessagesToDelete = append(startMessagesToDelete, startMessage)
			}
		} else {
			sender.logger.Info("Skipping start message whose time has not come", startMessage.LogDescription(), map[string]string{
				"current time": sender.timeProvider.Time().String(),
			})
		}
	}

	err := sender.store.SavePendingStartMessages(startMessagesToSave...)
	if err != nil {
		sender.logger.Error("Failed to save start messages to send", err)
		return err
	}
	err = sender.store.DeletePendingStartMessages(startMessagesToDelete...)
	if err != nil {
		sender.logger.Error("Failed to delete start messages", err)
		return err
	}

	return nil
}

func (sender *Sender) sendStopMessages(stopMessages map[string]models.PendingStopMessage) error {
	stopMessagesToSave := []models.PendingStopMessage{}
	stopMessagesToDelete := []models.PendingStopMessage{}

	for _, stopMessage := range stopMessages {
		if stopMessage.IsExpired(sender.timeProvider.Time()) {
			sender.logger.Info("Deleting expired stop message", stopMessage.LogDescription())
			stopMessagesToDelete = append(stopMessagesToDelete, stopMessage)
		} else if stopMessage.IsTimeToSend(sender.timeProvider.Time()) {
			shouldSend, isDuplicate, instanceToStop := sender.verifyStopMessageShouldBeSent(stopMessage)
			if shouldSend {
				messageToSend := models.StopMessage{
					AppGuid:       instanceToStop.AppGuid,
					AppVersion:    instanceToStop.AppVersion,
					InstanceIndex: instanceToStop.InstanceIndex,
					InstanceGuid:  stopMessage.InstanceGuid,
					IsDuplicate:   isDuplicate,
					MessageId:     stopMessage.MessageId,
				}
				err := sender.messageBus.Publish(sender.conf.SenderNatsStopSubject, string(messageToSend.ToJSON()))
				if err != nil {
					sender.logger.Error("Failed to send stop message", err, stopMessage.LogDescription())
					return err
				}
				if stopMessage.KeepAlive == 0 {
					sender.logger.Info("Deleting sent stop message with no keep alive", stopMessage.LogDescription())
					stopMessagesToDelete = append(stopMessagesToDelete, stopMessage)
				} else {
					stopMessage.SentOn = sender.timeProvider.Time().Unix()
					stopMessagesToSave = append(stopMessagesToSave, stopMessage)
				}
			} else {
				sender.logger.Info("Deleting stop message that will not be sent", stopMessage.LogDescription())
				stopMessagesToDelete = append(stopMessagesToDelete, stopMessage)
			}
		} else {
			sender.logger.Info("Skipping stop message whose time has not come", stopMessage.LogDescription())
		}
	}

	err := sender.store.SavePendingStopMessages(stopMessagesToSave...)
	if err != nil {
		sender.logger.Error("Failed to save stop messages to send", err)
		return err
	}
	err = sender.store.DeletePendingStopMessages(stopMessagesToDelete...)
	if err != nil {
		sender.logger.Error("Failed to delete stop messages", err)
		return err
	}

	return nil
}

func (sender *Sender) verifyStartMessageShouldBeSent(message models.PendingStartMessage) bool {
	appKey := sender.store.AppKey(message.AppGuid, message.AppVersion)
	app, found := sender.apps[appKey]

	if !found {
		sender.logger.Info("Skipping sending start message: app is no longer desired", message.LogDescription())
		return false
	}

	if !app.IsDesired() {
		sender.logger.Info("Skipping sending start message: app is no longer desired", message.LogDescription(), app.LogDescription())
		return false
	}

	if !app.IsIndexDesired(message.IndexToStart) {
		sender.logger.Info("Skipping sending start message: instance index is beyond the desired # of instances", message.LogDescription(), app.LogDescription())
		return false
	}

	if app.HasStartingOrRunningInstanceAtIndex(message.IndexToStart) {
		sender.logger.Info("Skipping sending start message: instance is already running", message.LogDescription(), app.LogDescription())
		return false
	}

	sender.logger.Info("Sending start message: instance is not running at desired index", message.LogDescription(), app.LogDescription())
	return true
}

func (sender *Sender) verifyStopMessageShouldBeSent(message models.PendingStopMessage) (bool, isDuplicate bool, instanceToStop models.InstanceHeartbeat) {
	appKey := sender.store.AppKey(message.AppGuid, message.AppVersion)
	app, found := sender.apps[appKey]

	if !found {
		sender.logger.Info("Skipping sending stop message: instance is no longer running", message.LogDescription())
		return false, false, models.InstanceHeartbeat{}
	}

	instanceToStop = app.InstanceWithGuid(message.InstanceGuid)

	if !app.IsDesired() {
		sender.logger.Info("Sending stop message: instance is running, app is no longer desired", message.LogDescription(), app.LogDescription())
		return true, false, instanceToStop
	}

	if !app.IsIndexDesired(instanceToStop.InstanceIndex) {
		sender.logger.Info("Sending stop message: index of instance to stop is beyond desired # of instances", message.LogDescription(), app.LogDescription())
		return true, false, instanceToStop
	}

	if len(app.StartingOrRunningInstancesAtIndex(instanceToStop.InstanceIndex)) > 1 {
		sender.logger.Info("Sending stop message: instance is a duplicate running at a desired index", message.LogDescription(), app.LogDescription())
		return true, true, instanceToStop
	}

	sender.logger.Info("Skipping sending stop message: instance is running on a desired index (and there are no other instances running at that index)", message.LogDescription(), app.LogDescription())
	return false, false, models.InstanceHeartbeat{}
}
