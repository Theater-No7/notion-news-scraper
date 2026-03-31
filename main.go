package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dstotijn/go-notion"
	"github.com/mmcdole/gofeed"
)

type Feed struct {
	URL       string `json:"url"`
	Category  string `json:"category"`
	MediaName string `json:"media_name"`
}

func main() {
	token := os.Getenv("NOTION_SECRET")
	dbID := os.Getenv("NOTION_DB_ID")

	if token == "" || dbID == "" {
		fmt.Println("❌ 環境変数が設定されていません")
		return
	}

	client := notion.NewClient(token)
	fp := gofeed.NewParser()

	file, _ := os.ReadFile("feeds.json")
	var feeds []Feed
	json.Unmarshal(file, &feeds)

	fmt.Println("🚀 インプット自動化システムを起動します...")

	for _, feed := range feeds {
		parsedFeed, err := fp.ParseURL(feed.URL)
		if err != nil {
			fmt.Printf("⚠️ %s の取得に失敗しました: %v\n", feed.MediaName, err)
			continue
		}

		fmt.Printf("\n▶ %s の最新記事をチェック中...\n", feed.MediaName)

		for i, item := range parsedFeed.Items {
			if i >= 3 {
				break
			}

			query := &notion.DatabaseQuery{
				Filter: &notion.DatabaseQueryFilter{
					Property: "Title",
					DatabaseQueryPropertyFilter: notion.DatabaseQueryPropertyFilter{
						Title: &notion.TextPropertyFilter{Equals: item.Title},
					},
				},
			}
			res, _ := client.QueryDatabase(context.Background(), dbID, query)
			if len(res.Results) > 0 {
				fmt.Printf("  ⏭ スキップ (既読): %s\n", item.Title)
				continue
			}

			pubDate := time.Now()
			if item.PublishedParsed != nil {
				pubDate = *item.PublishedParsed
			}
			notionDate := notion.NewDateTime(pubDate, false)
			link := item.Link

			_, err := client.CreatePage(context.Background(), notion.CreatePageParams{
				ParentType: notion.ParentTypeDatabase,
				ParentID:   dbID,
				DatabasePageProperties: &notion.DatabasePageProperties{
					"Title": notion.DatabasePageProperty{
						Title: []notion.RichText{
							{Text: &notion.Text{Content: item.Title}},
						},
					},
					"URL": notion.DatabasePageProperty{
						URL: &link,
					},
					"Category": notion.DatabasePageProperty{
						Select: &notion.SelectOptions{Name: feed.Category},
					},
					"Media": notion.DatabasePageProperty{
						Select: &notion.SelectOptions{Name: feed.MediaName},
					},
					"Date": notion.DatabasePageProperty{
						Date: &notion.Date{Start: notionDate},
					},
				},
			})

			if err != nil {
				fmt.Printf("  ❌ 書き込み失敗: %v\n", err)
			} else {
				fmt.Printf("  ✅ 保存成功: %s\n", item.Title)
			}
		}
	}
}
