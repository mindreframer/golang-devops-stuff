package actualstatelistener

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/helpers/metricsaccountant"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/store"
	"strconv"
	"sync"
	"time"

	"github.com/cloudfoundry/yagnats"
)

type ActualStateListener struct {
	logger            logger.Logger
	config            config.Config
	messageBus        yagnats.NATSClient
	store             store.Store
	timeProvider      timeprovider.TimeProvider
	storeUsageTracker metricsaccountant.UsageTracker
	metricsAccountant metricsaccountant.MetricsAccountant
	heartbeatsToSave  []models.Heartbeat
	heartbeatMutex    *sync.Mutex
}

func New(config config.Config,
	messageBus yagnats.NATSClient,
	store store.Store,
	storeUsageTracker metricsaccountant.UsageTracker,
	metricsAccountant metricsaccountant.MetricsAccountant,
	timeProvider timeprovider.TimeProvider,
	logger logger.Logger) *ActualStateListener {

	return &ActualStateListener{
		logger:            logger,
		config:            config,
		messageBus:        messageBus,
		store:             store,
		storeUsageTracker: storeUsageTracker,
		metricsAccountant: metricsAccountant,
		timeProvider:      timeProvider,
		heartbeatsToSave:  []models.Heartbeat{},
		heartbeatMutex:    &sync.Mutex{},
	}
}

func (listener *ActualStateListener) Start() {
	listener.messageBus.Subscribe("dea.advertise", func(message *yagnats.Message) {
		listener.bumpFreshness()
		listener.logger.Debug("Received dea.advertise")
	})

	listener.messageBus.Subscribe("dea.heartbeat", func(message *yagnats.Message) {
		listener.logger.Debug("Got a heartbeat")
		heartbeat, err := models.NewHeartbeatFromJSON([]byte(message.Payload))
		if err != nil {
			listener.logger.Error("Could not unmarshal heartbeat", err,
				map[string]string{
					"MessageBody": message.Payload,
				})
			return
		}

		listener.logger.Debug("Decoded the heartbeat")

		listener.metricsAccountant.IncrementReceivedHeartbeats()

		listener.heartbeatMutex.Lock()
		listener.heartbeatsToSave = append(listener.heartbeatsToSave, heartbeat)
		listener.logger.Info("Received a heartbeat", map[string]string{
			"Heartbeats Pending Save": strconv.Itoa(len(listener.heartbeatsToSave)),
		})
		listener.heartbeatMutex.Unlock()
	})

	go listener.syncHeartbeats()

	if listener.storeUsageTracker != nil {
		listener.storeUsageTracker.StartTrackingUsage()
		listener.measureStoreUsage()
	}
}

func (listener *ActualStateListener) syncHeartbeats() {
	syncInterval := time.NewTicker(listener.config.ListenerHeartbeatSyncInterval())

	for {
		listener.heartbeatMutex.Lock()
		heartbeatsToSave := listener.heartbeatsToSave
		listener.heartbeatsToSave = []models.Heartbeat{}
		listener.heartbeatMutex.Unlock()

		if len(heartbeatsToSave) > 0 {
			listener.logger.Info("Saving Heartbeats", map[string]string{
				"Heartbeats to Save": strconv.Itoa(len(heartbeatsToSave)),
			})

			t := time.Now()
			err := listener.store.SyncHeartbeats(heartbeatsToSave...)
			if err != nil {
				listener.logger.Error("Could not put instance heartbeats in store:", err)
				return
			}

			listener.bumpFreshness()
			listener.metricsAccountant.IncrementSavedHeartbeats(len(heartbeatsToSave))
			dt := time.Since(t)
			listener.logger.Info("Saved Heartbeats", map[string]string{
				"Heartbeats to Save": strconv.Itoa(len(heartbeatsToSave)),
				"Duration":           dt.String(),
			})
		}

		<-syncInterval.C
	}
}

func (listener *ActualStateListener) measureStoreUsage() {
	usage, _ := listener.storeUsageTracker.MeasureUsage()
	listener.metricsAccountant.TrackActualStateListenerStoreUsageFraction(usage)

	time.AfterFunc(3*time.Duration(listener.config.HeartbeatPeriod)*time.Second, func() {
		listener.measureStoreUsage()
	})
}

func (listener *ActualStateListener) bumpFreshness() {
	err := listener.store.BumpActualFreshness(listener.timeProvider.Time())
	if err != nil {
		listener.logger.Error("Could not update actual freshness", err)
	} else {
		listener.logger.Info("Bumped freshness")
	}
}
