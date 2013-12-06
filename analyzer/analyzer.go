package analyzer

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/store"
)

type Analyzer struct {
	store store.Store

	logger       logger.Logger
	timeProvider timeprovider.TimeProvider
	conf         *config.Config
}

func New(store store.Store, timeProvider timeprovider.TimeProvider, logger logger.Logger, conf *config.Config) *Analyzer {
	return &Analyzer{
		store:        store,
		timeProvider: timeProvider,
		logger:       logger,
		conf:         conf,
	}
}

func (analyzer *Analyzer) Analyze() error {
	err := analyzer.store.VerifyFreshness(analyzer.timeProvider.Time())
	if err != nil {
		analyzer.logger.Error("Store is not fresh", err)
		return err
	}

	apps, err := analyzer.store.GetApps()
	if err != nil {
		analyzer.logger.Error("Failed to fetch apps", err)
		return err
	}

	existingPendingStartMessages, err := analyzer.store.GetPendingStartMessages()
	if err != nil {
		analyzer.logger.Error("Failed to fetch pending start messages", err)
		return err
	}

	existingPendingStopMessages, err := analyzer.store.GetPendingStopMessages()
	if err != nil {
		analyzer.logger.Error("Failed to fetch pending stop messages", err)
		return err
	}

	allStartMessages := []models.PendingStartMessage{}
	allStopMessages := []models.PendingStopMessage{}
	allCrashCounts := []models.CrashCount{}

	for _, app := range apps {
		startMessages, stopMessages, crashCounts := newAppAnalyzer(app, analyzer.timeProvider.Time(), existingPendingStartMessages, existingPendingStopMessages, analyzer.logger, analyzer.conf).analyzeApp()
		for _, startMessage := range startMessages {
			allStartMessages = append(allStartMessages, startMessage)
		}
		for _, stopMessage := range stopMessages {
			allStopMessages = append(allStopMessages, stopMessage)
		}
		allCrashCounts = append(allCrashCounts, crashCounts...)
	}

	err = analyzer.store.SaveCrashCounts(allCrashCounts...)

	if err != nil {
		analyzer.logger.Error("Analyzer failed to save crash counts", err)
		return err
	}

	err = analyzer.store.SavePendingStartMessages(allStartMessages...)

	if err != nil {
		analyzer.logger.Error("Analyzer failed to enqueue start messages", err)
		return err
	}

	err = analyzer.store.SavePendingStopMessages(allStopMessages...)
	if err != nil {
		analyzer.logger.Error("Analyzer failed to enqueue stop messages", err)
		return err
	}

	return nil
}
