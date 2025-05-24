-- +goose Up
ALTER TABLE feeds
ADD last_fetched_at TIMESTAMP;

-- +goose DOWN
ALTER TABLE feeds
DROP COLUMN last_fetched_at;