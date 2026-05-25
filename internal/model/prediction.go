package model

import (
	"fmt"
	"time"
)

// Prediction представляет прогноз пользователя на матч
type Prediction struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	MatchID   int64     `json:"match_id"`
	Predicted string    `json:"predicted"` // формат "2:1"
	Actual    *string   `json:"actual"`    // реальный результат, если матч завершён
	Score     int       `json:"score"`     // очки за прогноз
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsValidFormat проверяет, что прогноз в формате "X:Y"
func IsValidFormat(score string) bool {
	// Простая валидация: две цифры через двоеточие
	// Можно усложнить с помощью regexp, если нужно
	_, _, err := ParseScore(score)
	return err == nil
}

// ParseScore парсит строку "2:1" в два целых числа
func ParseScore(score string) (home, away int, err error) {
	// Реализация парсинга
	// Возвращаем заглушку — реализуешь сам как упражнение
	return 0, 0, fmt.Errorf("not implemented")
}
