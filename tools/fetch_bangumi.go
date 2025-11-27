package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type AnimeInfo struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	NameCN  string   `json:"name_cn"`
	Year    int      `json:"year"`
	Date    string   `json:"date"`
	Summary string   `json:"summary"`
	Cover   string   `json:"cover"`
	Score   float64  `json:"score"`
	Tags    []string `json:"tags"`
}

type SearchRequest struct {
	Filter SearchFilter `json:"filter"`
}

type SearchFilter struct {
	Type    []int    `json:"type"`
	AirDate []string `json:"air_date"`
}

type SubjectData struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	NameCN  string  `json:"name_cn"`
	Date    string  `json:"date"`
	Image   string  `json:"image"`
	Summary string  `json:"summary"`
	Rating  struct {
		Score float64 `json:"score"`
	} `json:"rating"`
	Tags []struct {
		Name string `json:"name"`
	} `json:"tags"`
}

type SearchResponse struct {
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
	Data   []SubjectData `json:"data"`
}

var client = &http.Client{Timeout: 60 * time.Second}

func main() {
	os.MkdirAll("../data", 0755)
	allAnime := []AnimeInfo{}

	for year := 2000; year <= 2024; year++ {
		fmt.Printf("获取 %d 年...", year)
		animes := fetchByYear(year)
		allAnime = append(allAnime, animes...)
		fmt.Printf(" %d 部\n", len(animes))
	}

	data, _ := json.MarshalIndent(allAnime, "", "  ")
	os.WriteFile("../data/anime_db.json", data, 0644)
	fmt.Printf("\n总计 %d 部番剧，已保存到 data/anime_db.json\n", len(allAnime))
}

func fetchByYear(year int) []AnimeInfo {
	result := []AnimeInfo{}
	offset := 0
	limit := 50

	for {
		reqBody := SearchRequest{
			Filter: SearchFilter{
				Type:    []int{2},
				AirDate: []string{fmt.Sprintf(">=%d-01-01", year), fmt.Sprintf("<=%d-12-31", year)},
			},
		}

		jsonBody, _ := json.Marshal(reqBody)
		url := fmt.Sprintf("https://api.bgm.tv/v0/search/subjects?limit=%d&offset=%d", limit, offset)

		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "anime-site/1.0 (https://github.com/wukazis/anime)")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf(" 请求失败: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 429 {
			fmt.Printf(" 限流，等待...")
			time.Sleep(10 * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			fmt.Printf(" HTTP %d", resp.StatusCode)
			break
		}

		var searchResp SearchResponse
		if err := json.Unmarshal(body, &searchResp); err != nil {
			fmt.Printf(" 解析失败")
			break
		}

		for _, s := range searchResp.Data {
			tags := []string{}
			for i, t := range s.Tags {
				if i >= 5 { break }
				tags = append(tags, t.Name)
			}
			result = append(result, AnimeInfo{
				ID:      s.ID,
				Name:    s.Name,
				NameCN:  s.NameCN,
				Year:    year,
				Date:    s.Date,
				Summary: truncate(s.Summary, 200),
				Cover:   s.Image,
				Score:   s.Rating.Score,
				Tags:    tags,
			})
		}

		offset += limit
		if offset >= searchResp.Total {
			break
		}
		time.Sleep(time.Second) // 避免限流
	}

	return result
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
