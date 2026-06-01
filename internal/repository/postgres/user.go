package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MaximSunShine/TelegramBotFootboll/internal/model"
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

// GetByUsername находит пользователя по имени (username)
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	const query = `
		SELECT user_id, username, first_name, last_name, created_at, updated_at
		FROM users
		WHERE username = $1
	`
	var u model.User
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // не найдено — это не ошибка
		}
		return nil, fmt.Errorf("UserRepo.GetByUsername: %w", err)
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

// internal/repository/postgres/user.go

// ListActive возвращает список активных пользователей (кто делал прогнозы за последние 30 дней)
func (r *userRepository) ListActive(ctx context.Context, limit int) ([]*model.User, error) {
	const query = `
		SELECT DISTINCT u.user_id, u.username, u.first_name, u.last_name, u.created_at, u.updated_at
		FROM users u
		INNER JOIN predictions p ON u.user_id = p.user_id
		WHERE p.created_at > NOW() - INTERVAL '30 days'
		ORDER BY p.created_at DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("UserRepo.ListActive: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		err := rows.Scan(&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("UserRepo.ListActive scan: %w", err)
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}
