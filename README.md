# gator

`gator` is a CLI RSS feed aggregator. You register a user, follow RSS feeds, and a
long-running `agg` process periodically fetches every followed feed, parses the
posts, and stores them in a Postgres database. You can then browse the collected
posts from the comfort of your terminal.

It's built with Go, [sqlc](https://sqlc.dev), and PostgreSQL.

## Prerequisites

You'll need the following installed and on your `PATH`:

- **Go** 1.26 or newer — <https://go.dev/doc/install>
- **PostgreSQL** 12+ (any recent version works) — <https://www.postgresql.org/download/>
- **goose** (only for running database migrations) — install with:

  ```sh
  go install github.com/pressly/goose/v3/cmd/goose@latest
  ```

- **sqlc** (only if you plan to change SQL queries) — <https://sqlc.dev>

  The generated Go in `internal/database/` is checked in, so you only need sqlc
  if you modify `sql/queries/*.sql` and want to regenerate the bindings.

## Installation

Install the latest version straight from GitHub:

```sh
go install github.com/Filz0r/boot-blog-aggregator@latest
```

This fetches the source and installs the binary into your `$GOBIN` (which defaults
to `$GOPATH/bin`). Make sure that directory is on your `$PATH`:

```sh
# add to your shell profile (~/.bashrc, ~/.zshrc, etc.)
export PATH="$PATH:$(go env GOPATH)/bin"
```

The installed binary is named after the last path segment of the module — here
`boot-blog-aggregator`. If you'd rather call it `gator` (as the examples below do),
rename it after installing, or build it locally with the name you want:

```sh
# from a clone of this repo
go build -o gator
mv "$(go env GOPATH)/bin/boot-blog-aggregator" "$(go env GOPATH)/bin/gator"
```

The examples throughout this README use `gator` — substitute your binary's actual
name if you didn't rename it.

> Go produces statically compiled binaries — once `gator` is built you do **not**
> need the Go toolchain to run it. `go run .` is fine for development, but for
> regular use the installed binary is the way to go.

## Setup

### 1. Create a Postgres database

Create a database for gator to use (e.g. `gator`):

```sh
createdb gator
```

### 2. Apply the database migrations

The schema lives in `sql/schema/` as goose migrations. Apply them with goose,
pointing at the database URL you'll use as your config:

```sh
goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable" up
```

This creates the `users`, `feeds`, `feed_follows`, and `posts` tables.

### 3. Create the config file

gator reads its config from `~/.gatorconfig.json`. Create it with your Postgres
connection string:

```json
{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "current_user_name": null
}
```

- `db_url` — a standard Postgres connection string (`database/sql` / `lib/pq` format).
- `current_user_name` — set automatically when you log in or register; leave it
  `null` to start.

Make sure the connection string matches the database you migrated in step 2.

## Usage

First, register a user (this also logs you in):

```sh
gator register alice
```

Then try some commands:

```sh
# list every feed in the system
gator feeds

# add a feed (you automatically follow feeds you add)
gator addfeed "Hacker News" https://news.ycombinator.com/rss
gator addfeed "TechCrunch" https://techcrunch.com/feed/

# list the feeds you follow
gator following

# follow an existing feed by URL
gator follow https://techcrunch.com/feed/

# stop following a feed by URL
gator unfollow https://techcrunch.com/feed/
```

### Aggregating posts

Start the aggregator in the background. It takes a duration argument
(`time_between_reqs`) describing how often to fetch the next feed:

```sh
gator agg 1m
```

This prints `Collecting feed data every 1m0s` and then loops forever: each tick
it picks the feed that was fetched longest ago (or never fetched), marks it,
downloads the RSS, parses the posts, and saves any new ones to the database.
Duplicates (same post URL) are skipped silently.

Leave `gator agg` running in one terminal while you use gator in another. Stop
it at any time with **Ctrl+C**.

### Browsing posts

Once the aggregator has collected some posts, browse them:

```sh
# show the 2 most recent posts from feeds you follow (default limit is 2)
gator browse

# show more
gator browse 10
```

Posts are ordered newest-first.

## Commands

| Command                       | Login required | Description                                                        |
| ----------------------------- | :------------: | ------------------------------------------------------------------ |
| `register <username>`         |                | Create a new user and log in as them.                              |
| `login <username>`            |                | Log in as an existing user.                                        |
| `users`                       |                | List all registered users (marks the current one).                 |
| `reset`                       |                | Wipe the `users` table. Use with care.                             |
| `addfeed <name> <url>`        |       ✓        | Add a feed to the system and auto-follow it.                       |
| `feeds`                       |                | List every feed that has been added.                               |
| `follow <url>`                |       ✓        | Follow an existing feed by URL.                                    |
| `following`                   |       ✓        | List the feeds you currently follow.                               |
| `unfollow <url>`              |       ✓        | Stop following a feed by URL.                                      |
| `agg <time_between_reqs>`     |                | Run the aggregation loop (e.g. `agg 30s`, `agg 1m`, `agg 2h`).     |
| `browse [limit]`              |       ✓        | Show recent posts from followed feeds. `limit` defaults to `2`.    |

## Project layout

```
.
├── main.go                  # Entry point: wires up config, DB, and command handlers
├── go.mod                   # Module: github.com/Filz0r/boot-blog-aggregator
├── sql/
│   ├── schema/              # goose migrations (users, feeds, feed_follows, posts)
│   └── queries/             # sqlc queries, compiled into internal/database/
├── internal/
│   ├── commands/            # CLI command handlers and the scraper loop
│   ├── config/              # ~/.gatorconfig.json read/write
│   ├── database/            # sqlc-generated Go bindings for the queries above
│   └── system.go            # HTTP fetch + RSS/XML parsing helpers
```

The database layer is generated by sqlc from `sql/queries/*.sql`. If you change a
query, regenerate with `sqlc generate` from the repo root.

## Notes

- The scraper is polite on purpose: it fetches one feed per tick at the interval
  you choose, so pick a reasonable `time_between_reqs` (a minute or more is
  fine) and don't hammer third-party servers.
- Post timestamps are parsed best-effort into `published_at`. Feeds that omit a
  publish date or use an unusual format simply get a null `published_at` rather
  than crashing the scraper.