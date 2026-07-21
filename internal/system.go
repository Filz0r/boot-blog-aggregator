package internal

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"time"
)

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

func Pause() {
	fmt.Println("Press enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func FetchFeed(ctx context.Context, feedUrl string) (*RSSFeed, error) {
	result := RSSFeed{}
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", feedUrl, nil)
	if err != nil {
		return &result, fmt.Errorf("Error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "gator")

	res, err := client.Do(req)

	if err != nil {
		return &result, fmt.Errorf("Error making request: %w", err)
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)

	if err != nil {
		return &result, fmt.Errorf("Error reading body: %w", err)
	}

	err = xml.Unmarshal(data, &result)

	if err != nil {
		return &result, fmt.Errorf("Error unmarsheling the data: %w", err)
	}

	result.Channel.Title = html.UnescapeString(result.Channel.Title)
	result.Channel.Description = html.UnescapeString(result.Channel.Description)

	return &result, nil
}
