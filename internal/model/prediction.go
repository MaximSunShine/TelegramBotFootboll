package model

import (
	"fmt"
	"time"
)

// Prediction представляет прогноз пользователя на матч
type Prediction struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	MatchID       int64     `json:"match_id"`
	PredictedHome int       `json:"predicted_home"` // ← добавлено
	PredictedAway int       `json:"predicted_away"` // ← добавлено
	ActualHome    *int      `json:"actual_home"`    // ← добавлено (nullable)
	ActualAway    *int      `json:"actual_away"`    // ← добавлено
	Points        int       `json:"points"`         // ← добавлено
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Вложенный матч для удобного доступа в хендлерах
	Match *Match `json:"match,omitempty"`
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

// CalculateScore рассчитывает очки за прогноз
func (p *Prediction) CalculateScore() int {
	if p.ActualHome == nil || p.ActualAway == nil {
		return 0 // матч ещё не сыгран
	}

	// Точный счёт = 3 очка
	if p.PredictedHome == *p.ActualHome && p.PredictedAway == *p.ActualAway {
		return 3
	}

	// Угадан исход (победа/ничья/поражение) = 1 очко
	predictedResult := p.PredictedHome - p.PredictedAway
	actualResult := *p.ActualHome - *p.ActualAway

	if (predictedResult > 0 && actualResult > 0) ||
		(predictedResult < 0 && actualResult < 0) ||
		(predictedResult == 0 && actualResult == 0) {
		return 1
	}

	return 0
}
