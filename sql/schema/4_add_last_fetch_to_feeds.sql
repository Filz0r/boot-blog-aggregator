-- +goose up
alter table feeds
add last_fetched_at timestamp;
-- +goose down
alter table feeds drop column last_fetched_at;