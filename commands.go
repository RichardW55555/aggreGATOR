package main

import (
	"context"
	"fmt"
	"io"
	"time"
	"html"
	"net/http"
	"encoding/xml"
	"github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/richardw55555/aggreGATOR/internal/config"
	"github.com/richardw55555/aggreGATOR/internal/database"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	Name string
	Args []string
}

type commands struct {
	registeredCommands map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	handler, ok := c.registeredCommands[cmd.Name]
	if !ok {
		return fmt.Errorf("command not found")
	}

	return handler(s, cmd)
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.registeredCommands[name] = f
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	client := http.Client{
    	Timeout: 10 * time.Second,
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "gator")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	xmlData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var feed RSSFeed
	if err := xml.Unmarshal(xmlData, &feed); err != nil {
		return nil, err
	}

	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	for i := range feed.Channel.Item {
		feed.Channel.Item[i].Title = html.UnescapeString(feed.Channel.Item[i].Title)
		feed.Channel.Item[i].Description = html.UnescapeString(feed.Channel.Item[i].Description)
	}

	return &feed, nil
}

func scrapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background(),)
	if err != nil {
		return err
	}

	if err := s.db.MarkFeedFetched(
		context.Background(),
		feed.ID,
	); err != nil {
		return err
	}

	RSSFeed, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return err
	}

	for _, item := range RSSFeed.Channel.Item {
		layout := "Mon, 02 Jan 2006 15:04:05 -0700"
		PubDate, err := time.Parse(layout, item.PubDate)
		if err != nil {
			return err
		}

		post, err := s.db.CreatePost(
			context.Background(),
			database.CreatePostParams{
				ID:          uuid.New(),
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
				Title:       item.Title,
				Url:         item.Link,
				Description: item.Description,
				PublishedAt: PubDate,
				FeedID:      feed.ID,
			},
		)
		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				fmt.Printf("")
			} else {
				return err
			}
		}

		fmt.Printf("ID:          %s\n", post.ID)
		fmt.Printf("CreatedAt:   %v\n", post.CreatedAt)
		fmt.Printf("UpdatedAt:   %v\n", post.UpdatedAt)
		fmt.Printf("Title:       %s\n", post.Title)
		fmt.Printf("Url:         %s\n", post.Url)
		fmt.Printf("Description: %s\n", post.Description)
		fmt.Printf("PublishedAt: %v\n", post.PublishedAt)
		fmt.Printf("FeedID:      %s\n", post.FeedID)
	}

	return nil
}