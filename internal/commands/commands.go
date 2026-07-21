package commands

import (
	"boot-blog-aggregator/internal"
	"boot-blog-aggregator/internal/config"
	"boot-blog-aggregator/internal/database"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
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

func HanldleAgg(s *State, cmd Command) error {
	data, err := internal.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")

	if err != nil {
		return err
	}

	fmt.Println(data)

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
