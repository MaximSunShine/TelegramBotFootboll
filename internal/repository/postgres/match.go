package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MaximSunShine/TelegramBotFootboll/internal/model"
)

type MatchRepository struct {
	pool *pgxpool.Pool
}

func NewMatchRepo(pool *pgxpool.Pool) *MatchRepository {
	return &MatchRepository{pool: pool}
}

// CreateOrUpdate вставляет или обновляет матч (UPSERT по match_id из API)
func (r *MatchRepository) CreateOrUpdate(ctx context.Context, m *model.Match) error {
	const query = `
		INSERT INTO matches (match_id, home_team, away_team, start_time, status, home_score, away_score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (match_id) DO UPDATE SET
			home_team = EXCLUDED.home_team,
			away_team = EXCLUDED.away_team,
			start_time = EXCLUDED.start_time,
			status = EXCLUDED.status,
			home_score = EXCLUDED.home_score,
			away_score = EXCLUDED.away_score,
			updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query,
		m.ID, m.HomeTeam, m.AwayTeam, m.StartTime, m.Status, m.HomeScore, m.AwayScore,
	)
	if err != nil {
		return fmt.Errorf("MatchRepo.CreateOrUpdate: %w", err)
	}
	return nil
}

// GetUpcoming возвращает ближайшие матчи (следующие 24 часа)
func (r *MatchRepository) GetUpcoming(ctx context.Context, limit int) ([]*model.Match, error) {
	const query = `
		SELECT match_id, home_team, away_team, start_time, status, home_score, away_score, created_at, updated_at
		FROM matches
		WHERE status = 'scheduled' AND start_time > NOW()
		ORDER BY start_time ASC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("MatchRepo.GetUpcoming: %w", err)
	}
	defer rows.Close()

	var matches []*model.Match
	for rows.Next() {
		var m model.Match
		err := rows.Scan(&m.ID, &m.HomeTeam, &m.AwayTeam, &m.StartTime, &m.Status, &m.HomeScore, &m.AwayScore, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("MatchRepo.GetUpcoming scan: %w", err)
		}
		matches = append(matches, &m)
	}
	return matches, rows.Err()
}

// GetByID находит матч по ID из API
func (r *MatchRepository) GetByID(ctx context.Context, id int64) (*model.Match, error) {
	const query = `
		SELECT match_id, home_team, away_team, start_time, status, home_score, away_score, created_at, updated_at
		FROM matches
		WHERE match_id = $1
	`
	var m model.Match
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.HomeTeam, &m.AwayTeam, &m.StartTime, &m.Status, &m.HomeScore, &m.AwayScore, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("MatchRepo.GetByID: %w", err)
	}
	return &m, nil
}

// UpdateResults обновляет счёт завершённого матча
func (r *MatchRepository) UpdateResults(ctx context.Context, matchID int64, homeScore, awayScore int) error {
	const query = `
		UPDATE matches 
		SET status = 'finished', home_score = $1, away_score = $2, updated_at = NOW()
		WHERE match_id = $3 AND status != 'finished'
	`
	cmd, err := r.pool.Exec(ctx, query, homeScore, awayScore, matchID)
	if err != nil {
		return fmt.Errorf("MatchRepo.UpdateResults: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("match %d not found or already finished", matchID)
	}
	return nil
}
