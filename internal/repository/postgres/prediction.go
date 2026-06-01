package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MaximSunShine/TelegramBotFootboll/internal/model"
)

type PredictionRepository struct {
	pool *pgxpool.Pool
}

func NewPredictionRepo(pool *pgxpool.Pool) *PredictionRepository {
	return &PredictionRepository{pool: pool}
}

// Create сохраняет прогноз пользователя
func (r *PredictionRepository) Create(ctx context.Context, p *model.Prediction) error {
	const query = `
		INSERT INTO predictions (user_id, match_id, predicted_home, predicted_away, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (user_id, match_id) DO UPDATE SET
			predicted_home = EXCLUDED.predicted_home,
			predicted_away = EXCLUDED.predicted_away,
			updated_at = NOW()
		RETURNING id, points
	`
	err := r.pool.QueryRow(ctx, query, p.UserID, p.MatchID, p.PredictedHome, p.PredictedAway).
		Scan(&p.ID, &p.Points)
	if err != nil {
		return fmt.Errorf("PredictionRepo.Create: %w", err)
	}
	return nil
}

// internal/repository/postgres/prediction.go

// GetByMatch возвращает все прогнозы на конкретный матч
func (r *PredictionRepository) GetByMatch(ctx context.Context, matchID int64) ([]*model.Prediction, error) {
	const query = `
		SELECT p.id, p.user_id, p.match_id, p.predicted_home, p.predicted_away, 
		       p.actual_home, p.actual_away, p.points, p.created_at,
		       u.username, u.first_name, u.last_name
		FROM predictions p
		JOIN users u ON p.user_id = u.user_id
		WHERE p.match_id = $1
		ORDER BY p.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, matchID)
	if err != nil {
		return nil, fmt.Errorf("PredictionRepo.GetByMatch: %w", err)
	}
	defer rows.Close()

	var preds []*model.Prediction
	for rows.Next() {
		var p model.Prediction
		var username, firstName, lastName string
		err := rows.Scan(
			&p.ID, &p.UserID, &p.MatchID, &p.PredictedHome, &p.PredictedAway,
			&p.ActualHome, &p.ActualAway, &p.Points, &p.CreatedAt,
			&username, &firstName, &lastName,
		)
		if err != nil {
			return nil, fmt.Errorf("PredictionRepo.GetByMatch scan: %w", err)
		}
		// Опционально: заполняем данные пользователя, если нужно
		// p.User = &model.User{Username: username, ...}
		preds = append(preds, &p)
	}
	return preds, rows.Err()
}

// GetByUser возвращает прогнозы пользователя с данными матча
func (r *PredictionRepository) GetByUser(ctx context.Context, userID int64, limit int) ([]*model.Prediction, error) {
	const query = `
		SELECT p.id, p.user_id, p.match_id, p.predicted_home, p.predicted_away, 
		       p.actual_home, p.actual_away, p.points, p.created_at,
		       m.home_team, m.away_team, m.start_time, m.status, m.home_score, m.away_score
		FROM predictions p
		JOIN matches m ON p.match_id = m.match_id
		WHERE p.user_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("PredictionRepo.GetByUser: %w", err)
	}
	defer rows.Close()

	var preds []*model.Prediction
	for rows.Next() {
		var p model.Prediction
		var match model.Match
		err := rows.Scan(
			&p.ID, &p.UserID, &p.MatchID, &p.PredictedHome, &p.PredictedAway,
			&p.ActualHome, &p.ActualAway, &p.Points, &p.CreatedAt,
			&match.HomeTeam, &match.AwayTeam, &match.StartTime, &match.Status, &match.HomeScore, &match.AwayScore,
		)
		if err != nil {
			return nil, fmt.Errorf("PredictionRepo.GetByUser scan: %w", err)
		}
		p.Match = &match
		preds = append(preds, &p)
	}
	return preds, rows.Err()
}

// ListByUser возвращает историю прогнозов пользователя (алиас на GetByUser)
func (r *PredictionRepository) ListByUser(ctx context.Context, userID int64, limit int) ([]*model.Prediction, error) {
	// Можно просто делегировать GetByUser, если логика одинаковая:
	return r.GetByUser(ctx, userID, limit)
}

// CalculateAndSavePoints рассчитывает очки и сохраняет результат
// Вызывается, когда матч завершился и известны реальные счета
func (r *PredictionRepository) CalculateAndSavePoints(ctx context.Context, matchID int64, actualHome, actualAway int) error {
	// 1. Находим все прогнозы на этот матч
	const selectQuery = `SELECT id, user_id, predicted_home, predicted_away FROM predictions WHERE match_id = $1`
	rows, err := r.pool.Query(ctx, selectQuery, matchID)
	if err != nil {
		return fmt.Errorf("PredictionRepo.CalculateAndSavePoints select: %w", err)
	}
	defer rows.Close()

	// 2. Для каждого прогноза считаем очки и обновляем
	const updateQuery = `UPDATE predictions SET actual_home = $1, actual_away = $2, points = $3, updated_at = NOW() WHERE id = $4`

	for rows.Next() {
		var predID, userID, predHome, predAway int64
		if err := rows.Scan(&predID, &userID, &predHome, &predAway); err != nil {
			return fmt.Errorf("scan prediction: %w", err)
		}

		pred := &model.Prediction{
			PredictedHome: int(predHome),
			PredictedAway: int(predAway),
			ActualHome:    &actualHome,
			ActualAway:    &actualAway,
		}
		points := pred.CalculateScore()

		_, err := r.pool.Exec(ctx, updateQuery, actualHome, actualAway, points, predID)
		if err != nil {
			return fmt.Errorf("update prediction %d: %w", predID, err)
		}
	}
	return rows.Err()
}

// internal/repository/postgres/prediction.go

// GetByUserAndMatch находит прогноз конкретного пользователя на конкретный матч
// Возвращает (nil, nil), если прогноз не найден — это штатная ситуация
func (r *PredictionRepository) GetByUserAndMatch(ctx context.Context, userID, matchID int64) (*model.Prediction, error) {
	const query = `
		SELECT id, user_id, match_id, predicted_home, predicted_away, 
		       actual_home, actual_away, points, created_at, updated_at
		FROM predictions
		WHERE user_id = $1 AND match_id = $2
	`
	var p model.Prediction
	err := r.pool.QueryRow(ctx, query, userID, matchID).Scan(
		&p.ID, &p.UserID, &p.MatchID, &p.PredictedHome, &p.PredictedAway,
		&p.ActualHome, &p.ActualAway, &p.Points, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // прогноз не найден — это не ошибка
		}
		return nil, fmt.Errorf("PredictionRepo.GetByUserAndMatch: %w", err)
	}
	return &p, nil
}
