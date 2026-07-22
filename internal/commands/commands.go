package commands

import (
	"boot-blog-aggregator/internal"
	"boot-blog-aggregator/internal/config"
	"boot-blog-aggregator/internal/database"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type State struct {
	Config *config.Config
	Db     *database.Queries
}

type Command struct {
	Name string
	Args []string
}

type CommandFunc func(state *State, command Command) error

type Commands struct {
	Handlers map[string]CommandFunc
}

func (c Commands) Run(s *State, cmd Command) error {
	cb, ok := c.Handlers[cmd.Name]
	if !ok {
		return fmt.Errorf("This command does not exist, use the help command to check available commands")
	}
	err := cb(s, cmd)

	if err != nil {
		return err
	}
	return nil
}

func (c Commands) Register(name string, cb CommandFunc) {
	c.Handlers[name] = cb
}

func MiddlewareLoggedIn(
	handler func(s *State, cmd Command, user database.User) error,
) CommandFunc {
	return func(s *State, cmd Command) error {
		if s.Config.Username == nil {
			return fmt.Errorf("There's no user currently logged in")
		}

		user, err := s.Db.GetUser(context.Background(), *s.Config.Username)

		if err != nil {
			return fmt.Errorf("Error finding user in the database :%w\n", err)
		}
		return handler(s, cmd, user)
	}
}

func HandleLogin(s *State, cmd Command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("Insufficient arguments were provided, the <username> argument is required")
	}
	username := cmd.Args[0]

	user, err := s.Db.GetUser(context.Background(), username)

	if err != nil {
		return fmt.Errorf("Cannot login with a user that doesn't exist!")
	}

	err = s.Config.SetUser(&user.Name)

	if err != nil {
		return err
	}

	fmt.Println("Congratulations you are logged in as: ", *s.Config.Username)
	return nil
}

func HandleRegister(s *State, cmd Command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("Insufficient arguments were provided, the <username> argument is required")
	}
	usernameToAdd := cmd.Args[0]
	user, err := s.Db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      usernameToAdd,
	})

	if err != nil {
		return err
	}

	s.Config.SetUser(&user.Name)
	fmt.Printf("The user with the name of %s was created!\n", user.Name)
	return nil
}

func HandleReset(s *State, cmd Command) error {
	err := s.Db.ResetUsersTable(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Database reset was successfully!")
	return nil
}

func HandleUserList(s *State, cmd Command) error {
	users, err := s.Db.GetUsers(context.Background())

	if err != nil {
		return err
	}
	if len(users) == 0 {
		fmt.Println("There are no registered users! Please run the register command and try again!")
		return nil
	}
	for _, user := range users {
		fmt.Printf("\t* %s", user.Name)
		if user.Name == *s.Config.Username {
			fmt.Print(" (current)")
		}
		fmt.Print("\n")
	}
	return nil
}

func HandleAgg(s *State, cmd Command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("You can only pass a time_between_requests argument to this function!")
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])

	if err != nil {
		return fmt.Errorf("Error parsing duration time: %w", err)
	}
	ticker := time.NewTicker(timeBetweenRequests)

	fmt.Printf("Collecting feed data every %s\n", timeBetweenRequests.String())
	for ; ; <-ticker.C {
		err = scrapeFeeds(s, context.Background())
		if err != nil {
			fmt.Printf("There was a error processing the feed: %v\n", err)
		}
	}

	return nil
}

func HandleAddFeed(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 2 {
		return fmt.Errorf("You need to pass an <feed name> and <feed url> to this command!")
	}

	name := cmd.Args[0]
	url := cmd.Args[1]

	feed, err := s.Db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})

	if err != nil {
		return fmt.Errorf("Error creating the new feed: %w", err)
	}

	_, err = s.Db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})

	if err != nil {
		return fmt.Errorf("Error subscribing this user to the just created feed! %w", err)
	}

	fmt.Println(feed)

	return nil
}

func HandleListFeeds(s *State, cmd Command) error {
	if len(cmd.Args) != 0 {
		return fmt.Errorf("You cannot pass arguments to this command!")
	}

	feeds, err := s.Db.GetFeeds(context.Background())

	if err != nil {
		return fmt.Errorf("Error getting feeds from db: %w\n", err)
	}

	for _, feed := range feeds {
		fmt.Printf("\t Feed Name: %s\n\tFeed URL: %s\nAdded by: %s\n", feed.Name, feed.Url, feed.Username)
	}
	return nil
}

func HandleFollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("You need to pass the url of a feed in order to follow it!")
	}

	url := cmd.Args[0]

	feed, err := s.Db.GetFeedByUrl(context.Background(), url)

	if err != nil {
		return fmt.Errorf("Error finding feed url in the database: %w", err)
	}

	follow, err := s.Db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})

	fmt.Printf("Subscribed to %s with the endpoint of %s\n", follow.FeedName, follow.FeedName)
	return nil
}

func HandleFollowing(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 0 {
		return fmt.Errorf("You cannot pass arguments to this command!")
	}

	feeds, err := s.Db.GetFeedFollowsForUser(context.Background(), user.ID)

	if err != nil {
		return fmt.Errorf("Error finding feeds for this user in the database: %w", err)
	}

	for _, feed := range feeds {
		fmt.Printf("Feed Name:%s\nFeed URL: %s\nCreated By: %s\n", feed.FeedName, feed.FeedUrl, feed.CreatorName)
	}

	return nil
}

func HandleUnfollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("You need to pass the feed URL as a parameter")
	}

	url := cmd.Args[0]

	feed, err := s.Db.GetFeedByUrl(context.Background(), url)

	if err != nil {
		return fmt.Errorf("Cannot find the feed url in the database: %w", err)
	}
	unfollowed, err := s.Db.UnfollowFeed(context.Background(), database.UnfollowFeedParams{
		FeedID: feed.ID,
		UserID: user.ID,
	})

	if err != nil {
		return fmt.Errorf("Error unfollowing the feed %w", err)
	}
	fmt.Printf("Success unfollowing feed named %s with URL of %s\n", unfollowed.Name, unfollowed.Url)
	return nil
}

func HandleBrowse(s *State, cmd Command, user database.User) error {
	var limit int32
	if len(cmd.Args) >= 1 {
		l, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("Invalid limit was given: %w", err)
		}
		limit = int32(l)
	} else {
		limit = 2
	}
	posts, err := s.Db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		return fmt.Errorf("Error getting posts for this user: %w", err)
	}
	delimiter := strings.Repeat("#", 50)
	for i, post := range posts {
		fmt.Printf("%s\n", delimiter)
		fmt.Printf("Post:\t#%d (%s)\n", i, post.FeedName)
		fmt.Printf("Title:\t%s\n", post.Title)
		fmt.Printf("Description:\t%s\n", post.Description)
		fmt.Printf("Link:\t%s\n", post.Url)
		if post.PublishedAt.Valid {
			fmt.Printf("Title:\t%s\n", post.PublishedAt.Time.String())
		}
		fmt.Printf("%s\n\n", delimiter)

	}
	return nil
}

func scrapeFeeds(s *State, ctx context.Context) error {
	nextFeed, err := s.Db.GetNextFeedToFetch(ctx)
	if err != nil {
		return fmt.Errorf("Error getting next feed to fetch: %w", err)
	}
	marked, err := s.Db.MarkFeedFetched(ctx, database.MarkFeedFetchedParams{
		UpdatedAt: time.Now(),
		ID:        nextFeed.ID,
	})
	if err != nil {
		return fmt.Errorf("Error marking feed as fetched: %w", err)
	}
	fmt.Printf("Fetching data for feed %s (%s)\n", marked.Name, marked.Url)
	feedData, err := internal.FetchFeed(ctx, marked.Url)
	if err != nil {
		return fmt.Errorf("Error fetching feed data: %w", err)
	}
	count := 0
	for _, post := range feedData.Channel.Item {
		var pubDate sql.NullTime
		for _, layout := range []string{time.RFC1123Z, time.RFC1123, time.RFC3339, time.UnixDate} {
			if t, err := time.Parse(layout, post.PubDate); err == nil {
				pubDate = sql.NullTime{Time: t, Valid: true}
				break
			}
		}
		dbPost, err := s.Db.CreatePost(ctx, database.CreatePostParams{
			CreatedAt:   time.Now(),
			Title:       post.Title,
			Url:         post.Link,
			Description: post.Description,
			PublishedAt: pubDate,
			FeedID:      nextFeed.ID,
		})
		if err != nil {
			var pgErr *pq.Error
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				continue
			}
			fmt.Printf("Error saving new post %q: %v\n", post.Link, err)
			continue
		}
		fmt.Printf("Created new Post: %s\n", dbPost.Title)
		count++
	}
	if count == 0 {
		fmt.Println("No new posts were found!")
	} else {
		fmt.Printf("Added %v new posts from this feed!", count)
	}
	return nil
}
