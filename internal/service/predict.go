package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"TelegramBotFootboll/internal/model"
	"TelegramBotFootboll/internal/repository"
)

// Ошибки сервиса
var (
	ErrInvalidScoreFormat  = errors.New("invalid score format, expected 'X:Y'")
	ErrMatchNotFound       = errors.New("match not found")
	ErrMatchAlreadyStarted = errors.New("match has already started")
	ErrPredictionExists    = errors.New("prediction for this match already exists")
)

// predictService реализует PredictService
type predictService struct {
	userRepo       repository.UserRepository
	matchRepo      repository.MatchRepository
	predictionRepo repository.PredictionRepository
}

// NewPredictService создаёт новый сервис прогнозов
func NewPredictService(
	userRepo repository.UserRepository,
	matchRepo repository.MatchRepository,
	predictionRepo repository.PredictionRepository,
) *predictService {
	return &predictService{
		userRepo:       userRepo,
		matchRepo:      matchRepo,
		predictionRepo: predictionRepo,
	}
}

// SubmitPrediction обрабатывает новый прогноз от пользователя
func (s *predictService) SubmitPrediction(
	ctx context.Context,
	userID int64,
	matchID int64,
	predictedScore string,
) error {
	// 1. Валидация формата прогноза
	home, away, err := parseScore(predictedScore)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidScoreFormat, err)
	}

	// 2. Проверка, что матч существует и ещё не начался
	match, err := s.matchRepo.GetByID(ctx, matchID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMatchNotFound, err)
	}

	if match.StartedAt.Before(time.Now()) {
		return ErrMatchAlreadyStarted
	}

	// 3. Проверка, что пользователь ещё не делал прогноз на этот матч
	_, err = s.predictionRepo.GetByUserAndMatch(ctx, userID, matchID)
	if err == nil {
		// Прогноз уже есть
		return ErrPredictionExists
	}

	// 4. Создаём прогноз
	pred := &model.Prediction{
		UserID:    userID,
		MatchID:   matchID,
		Predicted: fmt.Sprintf("%d:%d", home, away),
		CreatedAt: time.Now(),
	}

	if err := s.predictionRepo.Create(ctx, pred); err != nil {
		return fmt.Errorf("failed to save prediction: %w", err)
	}

	return nil
}

// CalculateScore вычисляет очки за прогноз
// Правила:
// - Точный счёт: 3 очка
// - Правильный исход (победа/ничья/поражение): 1 очко
// - Неправильный: 0 очков
func (s *predictService) CalculateScore(predicted, actual string) (int, error) {
	pHome, pAway, err := parseScore(predicted)
	if err != nil {
		return 0, err
	}

	aHome, aAway, err := parseScore(actual)
	if err != nil {
		return 0, err
	}

	// Точное совпадение
	if pHome == aHome && pAway == aAway {
		return 3, nil
	}

	// Правильный исход
	predictedResult := getResult(pHome, pAway)
	actualResult := getResult(aHome, aAway)

	if predictedResult == actualResult {
		return 1, nil
	}

	return 0, nil
}

// GetUserPredictions возвращает последние прогнозы пользователя
func (s *predictService) GetUserPredictions(
	ctx context.Context,
	userID int64,
	limit int,
) ([]*model.Prediction, error) {
	return s.predictionRepo.ListByUser(ctx, userID, limit)
}

// parseScore парсит строку "2:1" в два целых числа
func parseScore(score string) (home, away int, err error) {
	parts := strings.Split(strings.TrimSpace(score), ":")
	if len(parts) != 2 {
		return 0, 0, ErrInvalidScoreFormat
	}

	home, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid home score: %w", err)
	}

	away, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid away score: %w", err)
	}

	return home, away, nil
}

// getResult возвращает результат матча: 1=победа дома, 0=ничья, -1=победа гостей
func getResult(home, away int) int {
	switch {
	case home > away:
		return 1
	case home < away:
		return -1
	default:
		return 0
	}
}
