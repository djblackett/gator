-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id) 
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetFeeds :many
SELECT feeds.name AS feed_name, feeds.url, users.name AS user_name 
FROM feeds
INNER JOIN users ON feeds.user_id=users.id; 

-- name: GetFeedByUrl :one
SELECT * FROM feeds
WHERE url = $1;

-- name: MarkFeedFetchedById :exec
UPDATE feeds
SET last_fetched_at = $2, updated_at = $3
WHERE id = $1;

-- name: GetNextFeedToFetch :one 
SELECT * FROM feeds
ORDER BY last_fetched_at NULLS FIRST
LIMIT 1;