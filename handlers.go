package main

import (
	"context"
	"fmt"
	"time"
	"strconv"
	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/richardw55555/aggreGATOR/internal/database"

)

func handlerLogin(s *state, cmd command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <name>", cmd.Name)
	}

	name := cmd.Args[0]
	
	if _, err := s.db.GetUser(context.Background(), name); err != nil {
		return fmt.Errorf("user does not exist")
	}
	
	if err := s.cfg.SetUser(name); err != nil {
		return err
	}

	fmt.Println("the username has been set")
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %v <name>", cmd.Name)
	}

	name := cmd.Args[0]

	user, err := s.db.CreateUser(
		context.Background(),
		database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			Name:      name,
		},
	)
	if err != nil {
		return err
	}

	if err := s.cfg.SetUser(name); err != nil {
		return err
	}

	fmt.Printf("user with id: %s registered\n", user.ID)
	return nil
}

func handlerReset(s *state, cmd command) error {
	return s.db.DeleteUsers(context.Background(),)
}

func handlerListUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background(),)
	if err != nil {
		return err
	}

	for _, user := range users {
		name := user.Name
		
		if name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", name)
			continue
		}
		
		fmt.Printf("* %s\n", name)
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		return fmt.Errorf("usage: %v <time_between_reqs>", cmd.Name)
	}
	
	time_between_reqs := cmd.Args[0]
	
	timeBetweenRequests, err := time.ParseDuration(time_between_reqs)
	if err != nil {
		return err
	}

	fmt.Printf("Collecting feeds every %v\n", timeBetweenRequests)
	fmt.Println("=====================================")
	scrapeFeeds(s)
	fmt.Println("=====================================")

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
		fmt.Println("=====================================")
	}
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 2 {
		return fmt.Errorf("usage: %s <name> <url>", cmd.Name)
	}

	name := cmd.Args[0]
	url := cmd.Args[1]

	feed, err := s.db.CreateFeed(
		context.Background(),
		database.CreateFeedParams{
			ID:        uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			Name:      name,
			Url:       url,
			UserID:    user.ID,
		},
	)
	if err != nil {
		return err
	}

	fmt.Println("Feed created successfully:")
	fmt.Println()
	fmt.Println("=====================================")
	fmt.Printf("ID:        %s\n", feed.ID)
	fmt.Printf("CreatedAt: %v\n", feed.CreatedAt)
	fmt.Printf("UpdatedAt: %v\n", feed.UpdatedAt)
	fmt.Printf("Name:      %s\n", feed.Name)
	fmt.Printf("Url:       %s\n", feed.Url)
	fmt.Printf("UserID:    %s\n", feed.UserID)
	fmt.Println("=====================================")

	_, err = s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			UserID:    user.ID,
			FeedID:    feed.ID,
		},
	)
	if err != nil {
		return err
	}

	fmt.Println("Feed followed successfully:")
	fmt.Println("=====================================")

	return nil
}

func handlerListFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background(),)
	if err != nil {
		return err
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	fmt.Printf("Found %d feeds:\n", len(feeds))
	fmt.Println("=====================================")
	for _, feed := range feeds {
		user, err := s.db.GetUserById(context.Background(), feed.UserID)
		if err != nil {
			return err
		}
		
		fmt.Printf("Name:     %s\n", feed.Name)
		fmt.Printf("Name:     %s\n", feed.Url)
		fmt.Printf("UserName: %s\n", user.Name)
		fmt.Println("=====================================")
	}

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <feed_url>", cmd.Name)
	}
	
	url := cmd.Args[0]
	
	feed, err := s.db.GetFeedByUrl(context.Background(), url)
	if err != nil {
		return err
	}

	_, err = s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			UserID:    user.ID,
			FeedID:    feed.ID,
		},
	)
	if err != nil {
		return err
	}

	fmt.Printf("FeedName: %s\n", feed.Name)
	fmt.Printf("CurrentUser: %s\n", s.cfg.CurrentUserName)

	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	feeds, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return err
	}

	for i, feed := range feeds {
		fmt.Printf("Feed %d: %s", i, feed.FeedName)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <feed_url>", cmd.Name)
	}
	
	url := cmd.Args[0]
	
	feed, err := s.db.GetFeedByUrl(context.Background(), url)
	if err != nil {
		return err
	}

	return s.db.DeleteFeedFollow(
		context.Background(),
		database.DeleteFeedFollowParams{
			UserID: user.ID,
			FeedID: feed.ID,
		},
	)
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	var limit int32
	if len(cmd.Args) == 0 {
		limit = int32(2)
	} else {
		num, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("usage: %s [limit]", cmd.Name)
		}

		limit = int32(num)
	}

	posts, err := s.db.GetPostsForUser(
		context.Background(),
		database.GetPostsForUserParams{
			ID:    user.ID,
			Limit: limit,
		},
	)

	if err != nil {
		return err
	}

	fmt.Println("=====================================")
	for _, post := range posts {
		fmt.Printf("ID:          %s\n", post.ID)
		fmt.Printf("CreatedAt:   %v\n", post.CreatedAt)
		fmt.Printf("UpdatedAt:   %v\n", post.UpdatedAt)
		fmt.Printf("Title:       %s\n", post.Title)
		fmt.Printf("Url:         %s\n", post.Url)
		fmt.Printf("Description: %s\n", post.Description)
		fmt.Printf("PublishedAt: %v\n", post.PublishedAt)
		fmt.Printf("FeedID:      %s\n", post.FeedID)
		fmt.Println("=====================================")
	}

	return nil
}