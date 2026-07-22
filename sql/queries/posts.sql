-- name: CreatePost :one
insert into posts(
        created_at,
        updated_at,
        title,
        url,
        description,
        published_at,
        feed_id
    )
values ($1, $1, $2, $3, $4, $5, $6)
returning *;
-- name: GetPostsForUser :many
with user_followed_feeds as (
    select *
    from feed_follows
    where feed_follows.user_id = $1
)
select posts.*,
    feeds.name as feed_name
from posts
    join user_followed_feeds on posts.feed_id = user_followed_feeds.feed_id
    inner join feeds on feeds.id = user_followed_feeds.feed_id
order by posts.published_at desc
limit $2;