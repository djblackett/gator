package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/djblackett/gator/internal/config"
	"github.com/djblackett/gator/internal/database"
	_ "github.com/lib/pq"
)

func main() {

	configuration := config.Read()
	// fmt.Println("config before writing in main:", configuration)

	state := config.State{
		Config: &configuration,
	}

	dbURL := state.Config.DbUrl

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("error connecting to the database")
		os.Exit(1)
	}

	dbQueries := database.New(db)
	state.Db = dbQueries

	commands := config.Commands{
		CommandsMap: make(map[string]func(*config.State, config.Command) error),
	}

	commands.Register("login", config.HandlerLogin)
	commands.Register("register", config.HandlerRegister)
	commands.Register("reset", config.HandlerDeleteUsers)
	commands.Register("users", config.HandlerGetUsers)
	commands.Register("agg", config.HandlerAgg)
	commands.Register("addfeed", config.MiddlewareLoggedIn(config.HandlerAddFeed))
	commands.Register("feeds", config.HandlerGetFeeds)
	commands.Register("follow", config.MiddlewareLoggedIn(config.HandlerFollow))
	commands.Register("following", config.MiddlewareLoggedIn(config.HandlerFollowing))
	commands.Register("unfollow", config.MiddlewareLoggedIn(config.HandlerUnfollow))
	commands.Register("browse", config.HandlerBrowse)

	args := os.Args
	if len(args) < 2 {
		fmt.Println("error: not enough arguments were supplied")
		os.Exit(1)
	}

	// if len(args) < 3 {
	// 	fmt.Println("error: a username is required")
	// 	os.Exit(1)
	// }

	command := args[1]

	// check if argument was supplied
	// var commandArgs []string
	// if len(args) == 3 {
	// 	commandArgs = []string{args[2]}
	// } else {
	// 	commandArgs = make([]string, 0)
	// }

	newCommand := config.Command{
		Name: command,
		Args: args[2:],
	}

	err = commands.Run(&state, newCommand)
	if err != nil {
		fmt.Println("error: could not run command", err)
		os.Exit(1)
	}

	// fmt.Println(config.Read())
}
