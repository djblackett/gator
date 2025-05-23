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
	dbQueries := database.New(db)
	state.Db = dbQueries

	commands := config.Commands{
		CommandsMap: make(map[string]func(*config.State, config.Command) error),
	}

	commands.Register("login", config.HandlerLogin)
	// configuration.SetUser("dave")

	args := os.Args
	if len(args) < 2 {
		fmt.Println("error: not enough arguments were supplied")
		os.Exit(1)
	}

	if len(args) < 3 {
		fmt.Println("error: a username is required")
		os.Exit(1)
	}

	command := args[1]
	username := args[2]

	newCommand := config.Command{
		Name: command,
		Args: []string{username},
	}

	err := commands.Run(&state, newCommand)
	if err != nil {
		fmt.Println("error: could not run command")
		os.Exit(1)
	}

	fmt.Println(config.Read())
}
