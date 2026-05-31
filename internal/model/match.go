package model

import (
	"fmt"
	"time"
)

// Score представляет счёт матча (для внешнего API)
type Score struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

// Match представляет футбольный матч
type Match struct {
	ID        int64     `json:"match_id"` // ID из внешнего API
	HomeTeam  string    `json:"home_team"`
	AwayTeam  string    `json:"away_team"`
	StartTime time.Time `json:"start_time"` // ← было StartAt, переименовали
	Status    string    `json:"status"`     // scheduled, live, finished
	HomeScore *int      `json:"home_score"` // nullable (матч ещё не сыгран)
	AwayScore *int      `json:"away_score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsFinished возвращает true, если матч завершён
func (m *Match) IsFinished() bool {
	return m.Status == "finished" && m.HomeScore != nil && m.AwayScore != nil
}

// Result возвращает строковое представление счёта "2:1" или пустую строку
func (m *Match) Result() string {
	if m.HomeScore == nil || m.AwayScore == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", *m.HomeScore, *m.AwayScore)
}
