package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MaximSunShine/TelegramBotFootboll/internal/bot"
	"github.com/MaximSunShine/TelegramBotFootboll/internal/model"
	"github.com/MaximSunShine/TelegramBotFootboll/internal/service"

	"github.com/MaximSunShine/TelegramBotFootboll/internal/config"
	"github.com/MaximSunShine/TelegramBotFootboll/internal/repository/postgres"
)

func main() {
	// 1. Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		slog.Error("❌ Failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Настраиваем логгер
	level := parseLogLevel(cfg.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	logger.Info("🚀 Application starting", "version", "0.1.0")

	// 3. Создаём корневой контекст с отменой по сигналу
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Подключаемся к базе данных
	pool, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("❌ Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("✅ Connected to PostgreSQL")

	// 5. Создаём репозитории
	userRepo := postgres.NewUserRepo(pool)
	matchRepo := postgres.NewMatchRepo(pool)
	predictionRepo := postgres.NewPredictionRepo(pool)

	// 6. Создаём сервисы
	predictSvc := service.NewPredictService(userRepo, matchRepo, predictionRepo)

	// 7. Создаём клиент внешнего API
	sstatsClient := service.NewSStatsClient(cfg.SStatsAPIBase, cfg.SStatsAPIKey)

	// 8. Запускаем фоновый синхронизатор
	syncWorker := service.NewSyncWorker(logger, matchRepo, predictionRepo, sstatsClient, 1*time.Hour)
	go syncWorker.Run(ctx) // не блокирует main

	// 7. Создаём и запускаем бота
	telegramBot, err := bot.New(cfg.TelegramBotToken, predictSvc, logger)
	if err != nil {
		logger.Error("❌ Failed to create bot", "error", err)
		os.Exit(1)
	}

	logger.Info("🤖 Bot created, starting update loop...")

	// 8. Запускаем бота с таймаутом на остановку
	if err := telegramBot.Run(ctx); err != nil {
		logger.Error("❌ Bot runtime error", "error", err)
		os.Exit(1)
	}

	// 9. Graceful shutdown
	logger.Info("👋 Graceful shutdown...")

	/*shutdownCtx*/
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info("✅ Application stopped")
}

// parseLogLevel конвертирует строку в slog.Level
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// stubPredictService — заглушка для компиляции, пока не реализованы все репозитории
// Удали этот код, когда реализуешь полноценные репозитории
type stubPredictService struct{}

func (s *stubPredictService) SubmitPrediction(ctx context.Context, userID, matchID int64, score string) error {
	return nil
}

func (s *stubPredictService) CalculateScore(predicted, actual string) (int, error) {
	return 0, nil
}

func (s *stubPredictService) GetUserPredictions(ctx context.Context, userID int64, limit int) ([]*model.Prediction, error) {
	return nil, nil
}
