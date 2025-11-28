package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

type SearchResult struct {
	AnimeName string `json:"anime_name"`
	AnimeID   int    `json:"anime_id"`
	Year      int    `json:"year"`
	Title     string `json:"title"`
	Magnet    string `json:"magnet"`
	PubDate   string `json:"pub_date"`
	Score     int    `json:"score"`
}

type RSS struct {
	Channel struct {
		Items []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title     string `xml:"title"`
	Link      string `xml:"link"`
	Enclosure struct {
		URL string `xml:"url,attr"`
	} `xml:"enclosure"`
}

var client = &http.Client{Timeout: 30 * time.Second}

func main() {
	// 读取现有结果
	data, _ := os.ReadFile("../data/dmhy_results.json")
	var results []SearchResult
	json.Unmarshal(data, &results)

	fixed := 0
	for i, r := range results {
		if r.Magnet == "" {
			fmt.Printf("修复: %s ... ", r.AnimeName)
			magnet := fetchMagnetFromSearch(r.AnimeName)
			if magnet != "" {
				results[i].Magnet = magnet
				fixed++
				fmt.Println("成功")
			} else {
				fmt.Println("失败")
			}
			time.Sleep(2 * time.Second)
		}
	}

	// 保存更新后的结果
	newData, _ := json.MarshalIndent(results, "", "  ")
	os.WriteFile("../data/dmhy_results.json", newData, 0644)

	fmt.Printf("\n修复了 %d 条\n", fixed)
}

func fetchMagnetFromSearch(keyword string) string {
	searchURL := fmt.Sprintf("https://share.dmhy.org/topics/rss/rss.xml?keyword=%s", url.QueryEscape(keyword))

	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var rss RSS
	xml.Unmarshal(body, &rss)

	// 筛选合集
	type scored struct {
		item  RSSItem
		score int
	}
	var candidates []scored

	for _, item := range rss.Channel.Items {
		title := strings.ToLower(item.Title)
		score := 0
		if strings.Contains(title, "合集") || strings.Contains(title, "全集") {
			score += 100
		}
		if strings.Contains(title, "简") || strings.Contains(title, "繁") {
			score += 50
		}
		if strings.Contains(title, "1080") {
			score += 30
		}
		candidates = append(candidates, scored{item, score})
	}

	if len(candidates) == 0 {
		return ""
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// 优先从 enclosure 获取
	for _, c := range candidates {
		if c.item.Enclosure.URL != "" {
			return c.item.Enclosure.URL
		}
	}

	// 从详情页获取
	for _, c := range candidates {
		if c.item.Link != "" {
			magnet := fetchMagnetFromPage(c.item.Link)
			if magnet != "" {
				return magnet
			}
		}
	}

	return ""
}

func fetchMagnetFromPage(pageURL string) string {
	req, _ := http.NewRequest("GET", pageURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// 匹配磁力链接
	re := regexp.MustCompile(`magnet:\?xt=urn:btih:[a-zA-Z0-9]+[^"'\s]*`)
	match := re.FindString(html)
	return match
}
