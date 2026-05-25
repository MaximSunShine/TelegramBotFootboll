DROP TRIGGER IF EXISTS update_predictions_updated_at ON predictions;
DROP TRIGGER IF EXISTS update_matches_updated_at ON matches;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS predictions;
DROP TABLE IF EXISTS matches;
DROP TABLE IF EXISTS users;