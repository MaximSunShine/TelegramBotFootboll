package bot

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/MaximSunShine/TelegramBotFootboll/internal/service"
)

// Bot представляет телеграм-бота
type Bot struct {
	api    *tgbotapi.BotAPI
	svc    service.PredictService
	logger *slog.Logger
}

// New создаёт нового бота
func New(token string, svc service.PredictService, logger *slog.Logger) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	return &Bot{
		api:    api,
		svc:    svc,
		logger: logger,
	}, nil
}

// Run запускает бота и обрабатывает обновления
func (b *Bot) Run(ctx context.Context) error {
	b.logger.Info("🤖 Bot starting", "username", b.api.Self.UserName)

	// Настройка обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("🛑 Bot stopped by context")
			return ctx.Err()

		case update, ok := <-updates:
			if !ok {
				b.logger.Error("❌ Updates channel closed")
				return fmt.Errorf("updates channel closed")
			}

			if update.Message != nil {
				b.handleMessage(ctx, update.Message)
			}
		}
	}
}

// handleMessage обрабатывает входящие сообщения
func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	// Игнорируем сообщения от ботов
	if msg.From.IsBot {
		return
	}

	// Команда /start
	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			b.handleStart(msg)
		case "predict":
			b.handlePredict(ctx, msg)
		case "help":
			b.handleHelp(msg)
		default:
			b.sendUnknownCommand(msg)
		}
		return
	}

	// Обработка обычного текста (можно расширить)
	b.sendEcho(msg)
}

// handleStart обрабатывает команду /start
func (b *Bot) handleStart(msg *tgbotapi.Message) {
	text := fmt.Sprintf(
		"👋 Привет, %s!\n\n"+
			"Я бот для прогнозов на футбол.\n"+
			"Используй команду /predict <match_id> <score> чтобы сделать прогноз.\n"+
			"Пример: /predict 123 2:1\n\n"+
			"/help — показать справку",
		msg.From.FirstName,
	)
	b.sendMessage(msg.Chat.ID, text)
}

// handlePredict обрабатывает команду /predict
func (b *Bot) handlePredict(ctx context.Context, msg *tgbotapi.Message) {
	// Парсим аргументы: /predict 123 2:1
	args := msg.CommandArguments() // "123 2:1"

	matchID, predictedScore, err := parsePredictArgs(args)
	if err != nil {
		b.sendError(msg.Chat.ID, fmt.Errorf("неверный формат команды. Используйте: /predict <id> <счёт>, например: /predict 123 2:1"))
		return
	}

	// Вызываем бизнес-логику
	if err := b.svc.SubmitPrediction(ctx, msg.From.ID, matchID, predictedScore); err != nil {
		b.logger.Error("failed to submit prediction", "user_id", msg.From.ID, "error", err)
		b.sendError(msg.Chat.ID, err)
		return
	}

	b.sendMessage(msg.Chat.ID, fmt.Sprintf("✅ Прогноз %s на матч #%d принят!", predictedScore, matchID))
}

// handleHelp обрабатывает команду /help
func (b *Bot) handleHelp(msg *tgbotapi.Message) {
	text := "📚 Справка:\n" +
		"/start — начать работу с ботом\n" +
		"/predict <match_id> <score> — сделать прогноз (пример: /predict 123 2:1)\n" +
		"/help — показать эту справку"
	b.sendMessage(msg.Chat.ID, text)
}

// sendUnknownCommand отправляет сообщение о неизвестной команде
func (b *Bot) sendUnknownCommand(msg *tgbotapi.Message) {
	b.sendMessage(msg.Chat.ID, "❌ Неизвестная команда. Используйте /help для списка команд.")
}

// sendEcho отправляет эхо-сообщение (для отладки)
func (b *Bot) sendEcho(msg *tgbotapi.Message) {
	b.sendMessage(msg.Chat.ID, fmt.Sprintf("🔁 Эхо: %s", msg.Text))
}

// sendError отправляет сообщение об ошибке пользователю
func (b *Bot) sendError(chatID int64, err error) {
	b.sendMessage(chatID, fmt.Sprintf("❌ Ошибка: %v", err))
}

// sendMessage отправляет текстовое сообщение
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("failed to send message", "chat_id", chatID, "error", err)
	}
}

// parsePredictArgs парсит аргументы команды /predict
func parsePredictArgs(args string) (matchID int64, score string, err error) {
	// Ожидаем: "123 2:1" или "123 2-1"
	parts := strings.Fields(args)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("ожидалось два аргумента: <match_id> <счёт>")
	}

	// Парсим ID матча
	matchID, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("неверный ID матча: %w", err)
	}

	// Валидируем формат счёта (2:1 или 2-1)
	score = strings.Replace(parts[1], "-", ":", 1)
	if matched, _ := regexp.MatchString(`^\d+:\d+$`, score); !matched {
		return 0, "", fmt.Errorf("неверный формат счёта, используйте 2:1")
	}

	return matchID, score, nil
}
