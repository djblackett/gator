package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/djblackett/gator/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
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

type RSSFeed struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Item        []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
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

func HandlerRegister(s *State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return errors.New("error: requires a username argument")
	}
	username := cmd.Args[0]
	user, err := s.Db.CreateUser(context.Background(), database.CreateUserParams{
		Name:      username,
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	if err != nil {
		fmt.Println("error: failed to create user", err)
		os.Exit(1)
	}
	fmt.Println("User created:", user)
	s.Config.SetUser(user.Name)
	return nil
}

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return errors.New("error: requires one argument")
	}

	username := cmd.Args[0]
	user, err := s.Db.GetUser(context.Background(), username)
	if err != nil {
		fmt.Println("user not in database")
		os.Exit(1)
	}

	s.Config.SetUser(user.Name)

	fmt.Printf("User %s has been set\n", user.Name)
	return nil
}

func HandlerDeleteUsers(s *State, cmd Command) error {
	err := s.Db.DeleteUsers(context.Background())
	if err != nil {
		fmt.Println("error: failed to delete users", err)
		os.Exit(1)
	}
	return nil
}

func HandlerGetUsers(s *State, cmd Command) error {
	users, err := s.Db.GetUsers(context.Background())
	if err != nil {
		fmt.Println("error: failed to get users")
		os.Exit(1)
	}

	for _, user := range users {
		if user.Name == s.Config.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Println("*", user.Name)
		}
	}
	return nil
}

func HandlerGetFeeds(s *State, cmd Command) error {
	feeds, err := s.Db.GetFeeds(context.Background())
	if err != nil {
		fmt.Println("error getting feeds", err)
		os.Exit(1)
	}

	for _, feed := range feeds {
		fmt.Println(feed.FeedName, feed.UserName, feed.Url)
	}
	return nil
}

func HandlerFollowing(s *State, cmd Command, user database.User) error {
	follows, err := s.Db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		throwErr("could not get feed follows", err)
	}

	for _, follow := range follows {
		fmt.Println(follow.FeedName)
	}
	return nil
}
func HandlerFollow(s *State, cmd Command, user database.User) error {

	feed, err := s.Db.GetFeedByUrl(context.Background(), cmd.Args[0])
	if err != nil {
		fmt.Println("error: could not get feed", err)
		os.Exit(1)
	}

	feedFollow, err := s.Db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		FeedID:    feed.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
	})

	if err != nil {
		fmt.Println("error following feed", err)
		os.Exit(1)
	}

	fmt.Println(feedFollow)
	return nil
}

func HandlerUnfollow(s *State, cmd Command, user database.User) error {

	feed, err := s.Db.GetFeedByUrl(context.Background(), cmd.Args[0])
	if err != nil {
		return err
	}

	err = s.Db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		fmt.Println("error deleting feed follow", err)
		return err
	}

	return nil
}

func HandlerAgg(s *State, cmd Command) error {
	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Collecting feeds every %v\n", timeBetweenRequests)
	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		err = ScrapeFeeds(s)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func HandlerBrowse(s *State, cmd Command) error {
	var limit int
	var err error
	if len(cmd.Args) > 0 {
		limit, err = strconv.Atoi(cmd.Args[0])
		if err != nil {
			throwErr("browse command accepts only integer args (default = 2)", err)
		}
	} else {
		limit = 2
	}
	user, err := s.Db.GetUser(context.Background(), s.Config.CurrentUserName)
	if err != nil {
		throwErr("cannot find user", err)
	}

	posts, err := s.Db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		throwErr("cannot get posts for user", err)
	}

	for _, post := range posts {

		fmt.Printf("Title: %s\n", post.Title)
		fmt.Printf("Published at: %v\n", post.PublishedAt)
		fmt.Printf("Description:\n%s", post.Description)
		fmt.Println("\n")

	}
	return nil
}
func HandlerAddFeed(s *State, cmd Command, user database.User) error {

	if len(cmd.Args) < 2 {
		fmt.Println("name and url args required")
		os.Exit(1)
	}
	url := cmd.Args[1]
	name := cmd.Args[0]

	feed, err := s.Db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Url:       url,
		Name:      name,
		UserID:    user.ID,
	})
	if err != nil {
		fmt.Println("error creating feed in database")
		os.Exit(1)
	}

	feedFollow, err := s.Db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		throwErr("could not create feedFollow", err)
	}

	fmt.Println(feed)
	fmt.Println("Feed follow created", feedFollow)
	return nil
}

func MiddlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(*State, Command) error {
	return func(s *State, cmd Command) error {
		currentUsername := s.Config.CurrentUserName
		user, err := s.Db.GetUser(context.Background(), currentUsername)
		if err != nil {
			fmt.Println("error getting user")
			return err
		}
		return handler(s, cmd, user)
	}
}

func ScrapeFeeds(s *State) error {
	feed, err := s.Db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	err = s.Db.MarkFeedFetchedById(context.Background(), database.MarkFeedFetchedByIdParams{
		ID:            feed.ID,
		UpdatedAt:     time.Now(),
		LastFetchedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return err
	}

	feed1, err := fetchFeed(context.Background(), feed.Url)

	if err != nil {
		return nil
	}
	for _, item := range feed1.Channel.Item {
		// Try to parse the pubDate string to time.Time
		pubDate, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			pubDate, err = time.Parse(time.RFC1123, item.PubDate)
			if err != nil {
				layout := "Mon, 2 Jan 2006 15:04:05 -0700"
				pubDate, err = time.Parse(layout, item.PubDate)
				if err != nil {
					layout := "2 Jan 2006 15:04:05 -0700"
					pubDate, err = time.Parse(layout, item.PubDate)

					if err != nil {
						fmt.Printf("Feed: %s, Title: %s - Warning: could not parse pubDate '%s': %v\n", feed.Name, item.Title, item.PubDate, err)
						// pubDate = time.Now()
					}
				}
			}
		}
		_, err = s.Db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			UpdatedAt:   time.Now(),
			CreatedAt:   time.Now(),
			Title:       item.Title,
			PublishedAt: pubDate,
			Description: item.Description,
			Url:         feed.Url,
			FeedID:      feed.ID,
		})

		if err != nil {
			if err, ok := err.(*pq.Error); ok {
				if err.Code.Name() == "unique_violation" {

				} else {
					fmt.Println("pq error:", err.Code.Name())
				}
			}
		}
	}
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

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		fmt.Println("error: could not create request", err)
		os.Exit(1)
	}

	req.Header.Set("User-Agent", "gator")

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("error: could not fetch rss feed", err)
		os.Exit(1)
	}

	resBytes, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("error: could not read bytes from res.Body", err)
		os.Exit(1)
	}
	var rss RSSFeed
	err = xml.Unmarshal(resBytes, &rss)

	if err != nil {
		fmt.Println("error: could not unmarshal xml data")
		os.Exit(1)
	}

	unescapedRssFeed := RSSFeed{
		Channel: Channel{
			Title:       html.UnescapeString(rss.Channel.Title),
			Description: html.UnescapeString(rss.Channel.Description),
			Link:        rss.Channel.Link,
			Item:        make([]RSSItem, 0, len(rss.Channel.Item)),
		},
	}

	for _, item := range rss.Channel.Item {
		unescapedRssFeed.Channel.Item = append(unescapedRssFeed.Channel.Item, RSSItem{
			Title:       html.UnescapeString(item.Title),
			Description: html.UnescapeString(item.Description),
			Link:        item.Link,
			PubDate:     item.PubDate,
		})
	}

	return &unescapedRssFeed, nil

}

func throwErr(message string, err error) {
	fmt.Println(message, err)
	os.Exit(1)
}
