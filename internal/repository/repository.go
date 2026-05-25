package repository

import (
	"context"

	"github.com/MaximSunShine/TelegramBotFootboll/internal/model"
)

// UserRepository определяет методы для работы с пользователями
type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
}

// MatchRepository определяет методы для работы с матчами
type MatchRepository interface {
	GetByID(ctx context.Context, id int64) (*model.Match, error)
	ListUpcoming(ctx context.Context, limit int) ([]*model.Match, error)
	ListFinished(ctx context.Context, limit int) ([]*model.Match, error)
	Create(ctx context.Context, match *model.Match) error
	Update(ctx context.Context, match *model.Match) error
}

// PredictionRepository определяет методы для работы с прогнозами
type PredictionRepository interface {
	Create(ctx context.Context, pred *model.Prediction) error
	GetByUserAndMatch(ctx context.Context, userID, matchID int64) (*model.Prediction, error)
	ListByUser(ctx context.Context, userID int64, limit int) ([]*model.Prediction, error)
	UpdateScore(ctx context.Context, id int64, score int, actual string) error
}
