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
	Keyword string       `json:"keyword,omitempty"`
	Filter  SearchFilter `json:"filter"`
}

type SearchFilter struct {
	Type    []int    `json:"type"`
	AirDate []string `json:"air_date,omitempty"`
}

type SearchResponse struct {
	Total int `json:"total"`
	Data  []struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		NameCN string `json:"name_cn"`
		Date   string `json:"date"`
		Image  string `json:"image"`
		Score  float64 `json:"score"`
		Tags   []struct {
			Name string `json:"name"`
		} `json:"tags"`
		Summary string `json:"summary"`
	} `json:"data"`
}

func main() {
	os.MkdirAll("../data", 0755)
	allAnime := []AnimeInfo{}

	for year := 2000; year <= 2024; year++ {
		fmt.Printf("获取 %d 年...", year)
		animes := fetchByYear(year)
		allAnime = append(allAnime, animes...)
		fmt.Printf(" %d 部\n", len(animes))
		time.Sleep(time.Second)
	}

	data, _ := json.MarshalIndent(allAnime, "", "  ")
	os.WriteFile("../data/anime_db.json", data, 0644)
	fmt.Printf("\n总计 %d 部番剧\n", len(allAnime))
}

func fetchByYear(year int) []AnimeInfo {
	result := []AnimeInfo{}
	offset := 0
	limit := 50

	for {
		reqBody := SearchRequest{
			Filter: SearchFilter{
				Type:    []int{2}, // 2 = 动画
				AirDate: []string{fmt.Sprintf(">=%d-01-01", year), fmt.Sprintf("<=%d-12-31", year)},
			},
		}

		jsonBody, _ := json.Marshal(reqBody)
		url := fmt.Sprintf("https://api.bgm.tv/v0/search/subjects?limit=%d&offset=%d", limit, offset)

		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "anime-site/1.0 (https://github.com/wukazis/anime)")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf(" 请求失败: %v", err)
			break
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Printf(" HTTP %d", resp.StatusCode)
			break
		}

		var searchResp SearchResponse
		if err := json.Unmarshal(body, &searchResp); err != nil {
			fmt.Printf(" 解析失败: %v", err)
			break
		}

		for _, s := range searchResp.Data {
			tags := []string{}
			for _, t := range s.Tags {
				tags = append(tags, t.Name)
			}
			result = append(result, AnimeInfo{
				ID:      s.ID,
				Name:    s.Name,
				NameCN:  s.NameCN,
				Year:    year,
				Date:    s.Date,
				Summary: s.Summary,
				Cover:   s.Image,
				Score:   s.Score,
				Tags:    tags,
			})
		}

		if len(searchResp.Data) < limit || offset+limit >= searchResp.Total {
			break
		}
		offset += limit
		time.Sleep(500 * time.Millisecond)
	}

	return result
}
