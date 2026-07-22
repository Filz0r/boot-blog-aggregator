-- +goose up
create table posts(
    id serial primary key,
    created_at timestamp not null,
    updated_at timestamp not null,
    title text not null,
    url text not null,
    description text not null,
    published_at timestamp,
    feed_id uuid not null,
    unique(url),
    constraint fk_feed foreign key (feed_id) references feeds(id) on delete cascade
);
-- +goose down
drop table posts;