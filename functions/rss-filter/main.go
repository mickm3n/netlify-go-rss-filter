package main

import (
	"encoding/xml"
	"fmt"
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

type Filter struct {
	Field string
	Value string
}

type FilterFunc func(*gofeed.Item, string) bool

var filterFunctions = map[string]FilterFunc{
	"title": func(item *gofeed.Item, keyword string) bool {
		return strings.Contains(strings.ToLower(item.Title), keyword)
	},
	"description": func(item *gofeed.Item, keyword string) bool {
		return strings.Contains(strings.ToLower(item.Description), keyword)
	},
	"author": func(item *gofeed.Item, keyword string) bool {
		return item.Author != nil && strings.Contains(strings.ToLower(item.Author.Name), keyword)
	},
	"content": func(item *gofeed.Item, keyword string) bool {
		return strings.Contains(strings.ToLower(item.Title), keyword) ||
			strings.Contains(strings.ToLower(item.Description), keyword) ||
			(item.Author != nil && strings.Contains(strings.ToLower(item.Author.Name), keyword))
	},
}

func parseAndFilterRSS(feedURL string, filters []Filter) (*RSS, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedURL)
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
		if passesAllFilters(item, filters) {
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

func passesAllFilters(item *gofeed.Item, filters []Filter) bool {
	for _, filter := range filters {
		if filterFunc, ok := filterFunctions[strings.ToLower(filter.Field)]; ok {
			if !filterFunc(item, strings.ToLower(filter.Value)) {
				return false
			}
		}
	}
	return true
}

func parseFilters(queryParams map[string]string) ([]Filter, error) {
	var filters []Filter
	for field, value := range queryParams {
		if field == "url" {
			continue
		}
		if _, ok := filterFunctions[strings.ToLower(field)]; !ok {
			return nil, fmt.Errorf("invalid filter field: %s", field)
		}
		filters = append(filters, Filter{Field: field, Value: value})
	}
	return filters, nil
}

func handler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	feedURL := request.QueryStringParameters["url"]
	if feedURL == "" {
		return &events.APIGatewayProxyResponse{StatusCode: 400, Body: "Missing 'url' parameter"}, nil
	}

	filters, err := parseFilters(request.QueryStringParameters)
	if err != nil {
		return &events.APIGatewayProxyResponse{StatusCode: 400, Body: "Invalid query parameters"}, nil
	}

	rss, err := parseAndFilterRSS(feedURL, filters)
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
