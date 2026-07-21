package main

import (
	"boot-blog-aggregator/internal/commands"
	"boot-blog-aggregator/internal/config"
	"boot-blog-aggregator/internal/database"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func main() {

	// Load the config file
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("FATAL: %s\n", err)
		os.Exit(1)
	}
	// Open db connection
	db, err := sql.Open("postgres", cfg.DbUrl)
	if err != nil {
		fmt.Printf("FATAL: %s\n", err)
		os.Exit(1)
	}

	dbQueries := database.New(db)

	// Init the state
	state := commands.State{
		Config: &cfg,
		Db:     dbQueries,
	}

	cmds := commands.Commands{Handlers: make(map[string]commands.CommandFunc)}

	// grap the cli args
	args := os.Args

	if len(args) < 2 {
		fmt.Println("FATAL: not enough arguments were provided")
		os.Exit(1)
	}

	cmds.Register("login", commands.HandleLogin)
	cmds.Register("register", commands.HandleRegister)
	cmds.Register("reset", commands.HandleReset)
	cmds.Register("users", commands.HandleUserList)
	cmds.Register("agg", commands.HanldleAgg)
	cmds.Register("addfeed", commands.MiddlewareLoggedIn(commands.HandleAddFeed))
	cmds.Register("feeds", commands.HandleListFeeds)
	cmds.Register("follow", commands.MiddlewareLoggedIn(commands.HandleFollow))
	cmds.Register("following", commands.MiddlewareLoggedIn(commands.HandleFollowing))

	// Run the commands
	cmd := commands.Command{
		Name: args[1],
		Args: args[2:],
	}

	err = cmds.Run(&state, cmd)

	if err != nil {
		fmt.Printf("FATAL: %s\n", err)
		os.Exit(1)
	}
}
