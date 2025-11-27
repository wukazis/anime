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

type AnimeInfo struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	NameCN string `json:"name_cn"`
	Year   int    `json:"year"`
}

type RSS struct {
	Channel struct {
		Items []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
	Enclosure struct {
		URL string `xml:"url,attr"`
	} `xml:"enclosure"`
}

type SearchResult struct {
	AnimeName string `json:"anime_name"`
	AnimeID   int    `json:"anime_id"`
	Year      int    `json:"year"`
	Title     string `json:"title"`
	Magnet    string `json:"magnet"`
	PubDate   string `json:"pub_date"`
	Score     int    `json:"score"` // 匹配得分
}

var client = &http.Client{Timeout: 30 * time.Second}

func main() {
	// 读取番剧数据
	data, _ := os.ReadFile("../data/anime_db.json")
	var animes []AnimeInfo
	json.Unmarshal(data, &animes)

	// 只处理评分高的热门番剧（可调整）
	fmt.Printf("共 %d 部番剧，开始搜索...\n", len(animes))

	var results []SearchResult
	notFound := []string{}

	for i, anime := range animes {
		name := anime.NameCN
		if name == "" {
			name = anime.Name
		}

		fmt.Printf("[%d/%d] 搜索: %s ... ", i+1, len(animes), name)

		result := searchDMHY(name, anime)
		if result != nil {
			results = append(results, *result)
			fmt.Printf("找到: %s\n", truncate(result.Title, 50))
		} else {
			notFound = append(notFound, name)
			fmt.Println("未找到合集")
		}

		time.Sleep(2 * time.Second) // 避免请求过快
	}

	// 保存结果
	saveResults(results, notFound)
}

func searchDMHY(keyword string, anime AnimeInfo) *SearchResult {
	// 使用 RSS 接口搜索
	searchURL := fmt.Sprintf("https://share.dmhy.org/topics/rss/rss.xml?keyword=%s", url.QueryEscape(keyword))

	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var rss RSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil
	}

	if len(rss.Channel.Items) == 0 {
		return nil
	}

	// 筛选和评分
	type scored struct {
		item  RSSItem
		score int
	}
	var candidates []scored

	for _, item := range rss.Channel.Items {
		title := strings.ToLower(item.Title)
		score := 0

		// 优先合集
		if strings.Contains(title, "合集") || strings.Contains(title, "全集") ||
			strings.Contains(title, "1-") || strings.Contains(title, "01-") {
			score += 100
		}

		// 优先有字幕组
		if strings.Contains(title, "简") || strings.Contains(title, "繁") ||
			strings.Contains(title, "字幕") || strings.Contains(title, "中文") {
			score += 50
		}

		// 优先高清
		if strings.Contains(title, "1080") {
			score += 30
		} else if strings.Contains(title, "720") {
			score += 20
		}

		// 优先 BDRip
		if strings.Contains(title, "bdrip") || strings.Contains(title, "bd") {
			score += 20
		}

		// 有磁力链接
		if item.Enclosure.URL != "" {
			score += 10
		}

		if score > 0 {
			candidates = append(candidates, scored{item, score})
		}
	}

	if len(candidates) == 0 {
		// 没有合集，取第一个有磁力的
		for _, item := range rss.Channel.Items {
			if item.Enclosure.URL != "" {
				return &SearchResult{
					AnimeName: keyword,
					AnimeID:   anime.ID,
					Year:      anime.Year,
					Title:     item.Title,
					Magnet:    item.Enclosure.URL,
					PubDate:   item.PubDate,
					Score:     0,
				}
			}
		}
		return nil
	}

	// 按得分排序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	best := candidates[0]
	magnet := best.item.Enclosure.URL
	if magnet == "" {
		magnet = extractMagnet(best.item.Link)
	}

	return &SearchResult{
		AnimeName: keyword,
		AnimeID:   anime.ID,
		Year:      anime.Year,
		Title:     best.item.Title,
		Magnet:    magnet,
		PubDate:   best.item.PubDate,
		Score:     best.score,
	}
}

func extractMagnet(pageURL string) string {
	// 如果需要从页面提取磁力链接
	resp, err := client.Get(pageURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	re := regexp.MustCompile(`magnet:\?xt=urn:btih:[a-zA-Z0-9]+`)
	match := re.FindString(string(body))
	return match
}

func saveResults(results []SearchResult, notFound []string) {
	// 保存找到的结果
	data, _ := json.MarshalIndent(results, "", "  ")
	os.WriteFile("../data/dmhy_results.json", data, 0644)

	// 保存未找到的
	os.WriteFile("../data/dmhy_notfound.txt", []byte(strings.Join(notFound, "\n")), 0644)

	// 生成磁力链接列表（方便批量下载）
	var magnets string
	for _, r := range results {
		if r.Magnet != "" {
			magnets += fmt.Sprintf("# %s (%d)\n%s\n\n", r.AnimeName, r.Year, r.Magnet)
		}
	}
	os.WriteFile("../data/magnets.txt", []byte(magnets), 0644)

	fmt.Printf("\n完成! 找到 %d 个, 未找到 %d 个\n", len(results), len(notFound))
	fmt.Println("结果保存到:")
	fmt.Println("  - data/dmhy_results.json (详细结果)")
	fmt.Println("  - data/magnets.txt (磁力链接列表)")
	fmt.Println("  - data/dmhy_notfound.txt (未找到的番剧)")
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "..."
	}
	return s
}
