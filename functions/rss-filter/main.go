package main

import (
	//"encoding/json"
	"encoding/xml"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/mmcdole/gofeed"
)

type Response events.APIGatewayProxyResponse

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Author      string `xml:"author,omitempty"`
	PubDate     string `xml:"pubDate,omitempty"`
}

func parseAndFilterRSS(url, filterField, filterKeyword string) (*RSS, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, err
	}

	rss := &RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       feed.Title,
			Link:        feed.Link,
			Description: feed.Description,
		},
	}

	for _, item := range feed.Items {
		if passesFilter(item, filterField, filterKeyword) {
			rssItem := Item{
				Title:       item.Title,
				Link:        item.Link,
				Description: item.Description,
				Author:      item.Author.Name,
				PubDate:     item.Published,
			}
			rss.Channel.Items = append(rss.Channel.Items, rssItem)
		}
	}

	return rss, nil
}

func passesFilter(item *gofeed.Item, filterField, filterKeyword string) bool {
	if filterKeyword == "" {
		return true
	}

	lowerKeyword := strings.ToLower(filterKeyword)
	switch strings.ToLower(filterField) {
	case "title":
		return strings.Contains(strings.ToLower(item.Title), lowerKeyword)
	case "description":
		return strings.Contains(strings.ToLower(item.Description), lowerKeyword)
	case "author":
		return item.Author != nil && strings.Contains(strings.ToLower(item.Author.Name), lowerKeyword)
	case "content":
		return strings.Contains(strings.ToLower(item.Title), lowerKeyword) ||
			strings.Contains(strings.ToLower(item.Description), lowerKeyword) ||
			(item.Author != nil && strings.Contains(strings.ToLower(item.Author.Name), lowerKeyword))
	default:
		return true
	}
}

func handler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	url := request.QueryStringParameters["url"]
	filterField := request.QueryStringParameters["field"]
	filterKeyword := request.QueryStringParameters["filter"]

	if url == "" {
		return &events.APIGatewayProxyResponse{StatusCode: 400, Body: "Missing 'url' parameter"}, nil
	}

	rss, err := parseAndFilterRSS(url, filterField, filterKeyword)
	if err != nil {
		return &events.APIGatewayProxyResponse{StatusCode: 500, Body: err.Error()}, nil
	}

	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return &events.APIGatewayProxyResponse{StatusCode: 500, Body: "Error encoding XML"}, nil
	}

	xmlString := xml.Header + string(output)

	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/rss+xml",
		},
		Body: xmlString,
	}, nil
}

func main() {
	lambda.Start(handler)
}
