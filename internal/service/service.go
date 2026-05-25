package service

import (
	"context"

	"github.com/MaximSunShine/TelegramBotFootboll/internal/model"
)

// PredictService определяет бизнес-логику для работы с прогнозами
type PredictService interface {
	SubmitPrediction(ctx context.Context, userID int64, matchID int64, predictedScore string) error
	CalculateScore(predicted, actual string) (int, error)
	GetUserPredictions(ctx context.Context, userID int64, limit int) ([]*model.Prediction, error)
}
