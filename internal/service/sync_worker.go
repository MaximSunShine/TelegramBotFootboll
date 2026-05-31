package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MaximSunShine/TelegramBotFootboll/internal/repository"
)

// SyncWorker — фоновая задача для синхронизации матчей и расчёта очков
type SyncWorker struct {
	logger       *slog.Logger
	matchRepo    repository.MatchRepository
	predRepo     repository.PredictionRepository
	sstatsClient *SStatsClient
	interval     time.Duration
}

func NewSyncWorker(
	logger *slog.Logger,
	matchRepo repository.MatchRepository,
	predRepo repository.PredictionRepository,
	sstatsClient *SStatsClient,
	interval time.Duration,
) *SyncWorker {
	return &SyncWorker{
		logger:       logger,
		matchRepo:    matchRepo,
		predRepo:     predRepo,
		sstatsClient: sstatsClient,
		interval:     interval,
	}
}

// Run запускает цикл синхронизации (блокирующий, запускать в горутине)
func (w *SyncWorker) Run(ctx context.Context) {
	w.logger.Info("🔄 SyncWorker started", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("🛑 SyncWorker stopped")
			return
		case <-ticker.C:
			if err := w.syncOnce(ctx); err != nil {
				w.logger.Error("❌ SyncWorker iteration failed", "error", err)
			}
		}
	}
}

func (w *SyncWorker) syncOnce(ctx context.Context) error {
	w.logger.Debug("🔄 Starting sync iteration")

	// 1. Загружаем ближайшие матчи из внешнего API
	externalMatches, err := w.sstatsClient.GetUpcomingMatches(ctx)
	if err != nil {
		return fmt.Errorf("fetch upcoming matches: %w", err)
	}
	w.logger.Info("📥 Fetched matches from API", "count", len(externalMatches))

	// 2. Сохраняем/обновляем в БД
	for _, m := range externalMatches {
		if err := w.matchRepo.CreateOrUpdate(ctx, m); err != nil {
			w.logger.Warn("⚠️ Failed to save match", "match_id", m.ID, "error", err)
		}
	}

	// 3. Проверяем завершённые матчи и считаем очки
	// (упрощённо: запрашиваем результаты для всех матчей со статусом 'live')
	finishedMatches, err := w.matchRepo.GetUpcoming(ctx, 100) // в реальности — отдельный метод GetByStatus('finished')
	if err != nil {
		return fmt.Errorf("get finished matches: %w", err)
	}

	for _, m := range finishedMatches {
		if m.Status == "finished" && m.HomeScore != nil && m.AwayScore != nil {
			if err := w.predRepo.CalculateAndSavePoints(ctx, m.ID, *m.HomeScore, *m.AwayScore); err != nil {
				w.logger.Warn("⚠️ Failed to calculate points", "match_id", m.ID, "error", err)
			} else {
				w.logger.Info("✅ Calculated points for match", "match_id", m.ID)
			}
		}
	}

	w.logger.Debug("✅ Sync iteration completed")
	return nil
}
