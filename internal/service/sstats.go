package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MaximSunShine/TelegramBotFootboll/internal/model"
)

// SStatsClient — клиент для внешнего API с футбольной статистикой
type SStatsClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewSStatsClient(baseURL, apiKey string) *SStatsClient {
	return &SStatsClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetUpcomingMatches загружает ближайшие матчи из внешнего API
func (c *SStatsClient) GetUpcomingMatches(ctx context.Context) ([]*model.Match, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/matches/upcoming", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var matches []*model.Match
	if err := json.NewDecoder(resp.Body).Decode(&matches); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return matches, nil
}

// GetMatchResults загружает результаты завершённых матчей
func (c *SStatsClient) GetMatchResults(ctx context.Context, matchIDs []int64) (map[int64]model.Score, error) {
	// Упрощённая реализация: один запрос на матч
	// В продакшене лучше делать батч-запрос
	results := make(map[int64]model.Score)

	for _, id := range matchIDs {
		req, err := http.NewRequestWithContext(ctx, "GET",
			fmt.Sprintf("%s/matches/%d/result", c.baseURL, id), nil)
		if err != nil {
			continue // пропускаем с ошибкой, логируем позже
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue
		}

		var result struct {
			HomeScore int `json:"home_score"`
			AwayScore int `json:"away_score"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			results[id] = model.Score{Home: result.HomeScore, Away: result.AwayScore}
		}
		resp.Body.Close()
	}
	return results, nil
}
