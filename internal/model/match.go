package model

import (
	"fmt"
	"time"
)

// Match представляет футбольный матч
type Match struct {
	ID        int64     `json:"id"`
	HomeTeam  string    `json:"home_team"`
	AwayTeam  string    `json:"away_team"`
	League    string    `json:"league"`
	StartedAt time.Time `json:"started_at"`
	Status    string    `json:"status"` // scheduled, live, finished
	HomeScore *int      `json:"home_score"`
	AwayScore *int      `json:"away_score"`
	SStatsID  int64     `json:"sstats_id"` // ID во внешнем API
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
