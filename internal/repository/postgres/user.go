package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"TelegramBotFootboll/internal/model"
)

// userRepository реализует UserRepository для PostgreSQL
type userRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepo создаёт новый репозиторий пользователей
func NewUserRepo(pool *pgxpool.Pool) *userRepository {
	return &userRepository{pool: pool}
}

// GetByID находит пользователя по ID
func (r *userRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	const query = `
		SELECT user_id, username, first_name, last_name, created_at, updated_at
		FROM users
		WHERE user_id = $1
	`

	var u model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &u, nil
}

// Create создаёт нового пользователя (UPSERT)
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	const query = `
		INSERT INTO users (user_id, username, first_name, last_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE
			SET username = EXCLUDED.username,
			    first_name = EXCLUDED.first_name,
			    last_name = EXCLUDED.last_name,
			    updated_at = NOW()
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Username, user.FirstName, user.LastName, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create/update user: %w", err)
	}

	return nil
}

// Update обновляет данные пользователя
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	const query = `
		UPDATE users
		SET username = $2, first_name = $3, last_name = $4, updated_at = NOW()
		WHERE user_id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		user.ID, user.Username, user.FirstName, user.LastName,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found for update: %d", user.ID)
	}

	return nil
}
