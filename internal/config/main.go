package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/djblackett/gator/internal/database"
)

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name,omitempty"`
}

type State struct {
	Db     *database.Queries
	Config *Config
}

type Command struct {
	Name string
	Args []string
}

type Commands struct {
	CommandsMap map[string]func(*State, Command) error
}

func (c *Commands) Run(s *State, cmd Command) error {
	function, ok := c.CommandsMap[cmd.Name]
	if !ok {
		return errors.New("error: command not in map")
	}
	err := function(s, cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c *Commands) Register(name string, f func(*State, Command) error) {
	c.CommandsMap[name] = f
}

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return errors.New("error: requires one argument")
	}

	username := cmd.Args[0]
	s.Config.SetUser(username)

	fmt.Println("User has been set")
	return nil
}

const configFileName = "/.gatorconfig.json"

func (c Config) SetUser(name string) {

	c = Read()

	filepath, err := getConfigFilePath()
	if err != nil {
		fmt.Println("Error getting config file path:", err)
		return
	}

	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	check(err)
	defer f.Close()
	c.CurrentUserName = name
	encoder := json.NewEncoder(f)
	err = encoder.Encode(c)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
	fmt.Println("Config written to file:", filepath)
}
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func Read() Config {

	filepath, err := getConfigFilePath()
	if err != nil {
		fmt.Println("Error getting config file path:", err)
		return Config{}
	}

	f, err := os.Open(filepath)
	check(err)

	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return Config{}
	}

	// fmt.Printf("text: %v\n", string(content))

	var config Config

	err = json.Unmarshal(content, &config)
	if err != nil {
		fmt.Println("Error decoding JSON to Config{}:", err)
	}
	return config
}

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("User home directory not set", err)
		return "", err
	}

	filepath := home + configFileName
	return filepath, nil
}
