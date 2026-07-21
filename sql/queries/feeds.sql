-- name: CreateFeed :one
insert into feeds(id, created_at, updated_at, name, url, user_id)
values ($1, $2, $3, $4, $5, $6)
returning *;
-- name: GetFeedsCreatedByUser :many
select *
from feeds
where user_id = $1;
-- name: GetFeeds :many
select feeds.name,
    feeds.url,
    users.name as username
from feeds
    INNER JOIN users on feeds.user_id = users.id;
-- name: GetFeedByUrl :one
select *
from feeds
where feeds.url = $1;
-- name: CreateFeedFollow :one
with inserted_feed AS (
    insert into feed_follows(created_at, updated_at, user_id, feed_id)
    values($1, $2, $3, $4)
    returning *
)
select inserted_feed.*,
    feeds.name as feed_name,
    feeds.url as feed_url,
    users.name as user_name
from inserted_feed
    inner join users on inserted_feed.user_id = users.id
    inner join feeds on inserted_feed.feed_id = feeds.id;
-- name: GetFeedFollowsForUser :many
select feeds.id,
    feeds.name as feed_name,
    feeds.url as feed_url,
    creator.name as creator_name
from feed_follows
    inner join feeds on feed_follows.feed_id = feeds.id
    inner join users as creator on feeds.user_id = creator.id
where feed_follows.user_id = $1;
-- name: UnfollowFeed :one
with deleted as (
    delete from feed_follows
    where feed_follows.feed_id = $1
        and feed_follows.user_id = $2
    returning feed_id
)
select feeds.id,
    feeds.name,
    feeds.url
from deleted
    inner join feeds on feeds.id = deleted.feed_id;