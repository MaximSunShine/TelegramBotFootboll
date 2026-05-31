// internal/repository/repository.go

package repository

import (
	"context"

	"github.com/MaximSunShine/TelegramBotFootboll/internal/model"
)

// UserRepository — работа с пользователями
type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	ListActive(ctx context.Context, limit int) ([]*model.User, error)
}

// MatchRepository — работа с матчами
type MatchRepository interface {
	GetByID(ctx context.Context, id int64) (*model.Match, error)
	GetUpcoming(ctx context.Context, limit int) ([]*model.Match, error)               // ← ДОБАВЛЕНО
	CreateOrUpdate(ctx context.Context, match *model.Match) error                     // ← ДОБАВЛЕНО
	UpdateResults(ctx context.Context, matchID int64, homeScore, awayScore int) error // ← ДОБАВЛЕНО
}

// PredictionRepository — работа с прогнозами
type PredictionRepository interface {
	Create(ctx context.Context, prediction *model.Prediction) error
	GetByUser(ctx context.Context, userID int64, limit int) ([]*model.Prediction, error)
	GetByMatch(ctx context.Context, matchID int64) ([]*model.Prediction, error)

	// ← ДОБАВИТЬ ЭТИ ДВА МЕТОДА:
	GetByUserAndMatch(ctx context.Context, userID, matchID int64) (*model.Prediction, error)
	ListByUser(ctx context.Context, userID int64, limit int) ([]*model.Prediction, error)

	CalculateAndSavePoints(ctx context.Context, matchID int64, actualHome, actualAway int) error
}

// Compile-time interface checks
/*var _ UserRepository = (*postgres.UserRepository)(nil)
var _ MatchRepository = (*postgres.MatchRepository)(nil)
var _ PredictionRepository = (*postgres.PredictionRepository)(nil)
*/
