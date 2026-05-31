-- Таблица матчей (синхронизируется с внешним API)
CREATE TABLE IF NOT EXISTS matches (
    match_id BIGINT PRIMARY KEY,          -- ID из SStats API
    home_team TEXT NOT NULL,
    away_team TEXT NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'scheduled', -- scheduled, live, finished
    home_score INTEGER,
    away_score INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_matches_start_time ON matches(start_time);
CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status);

-- Таблица прогнозов (связь users ↔ matches)
CREATE TABLE IF NOT EXISTS predictions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    match_id BIGINT NOT NULL REFERENCES matches(match_id) ON DELETE CASCADE,
    predicted_home INTEGER NOT NULL,
    predicted_away INTEGER NOT NULL,
    actual_home INTEGER,
    actual_away INTEGER,
    points INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Один прогноз на матч от одного пользователя
    UNIQUE(user_id, match_id)
);

CREATE INDEX IF NOT EXISTS idx_predictions_user ON predictions(user_id);
CREATE INDEX IF NOT EXISTS idx_predictions_match ON predictions(match_id);
CREATE INDEX IF NOT EXISTS idx_predictions_points ON predictions(points DESC);

-- Триггер для авто-обновления updated_at (как в users)
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_matches_updated_at 
    BEFORE UPDATE ON matches 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_predictions_updated_at 
    BEFORE UPDATE ON predictions 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();