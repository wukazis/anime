package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type SearchResult struct {
	AnimeName string `json:"anime_name"`
	AnimeID   int    `json:"anime_id"`
	Year      int    `json:"year"`
	Title     string `json:"title"`
	Magnet    string `json:"magnet"`
	Score     int    `json:"score"`
}

type VerifyIssue struct {
	AnimeID int `json:"anime_id"`
}

func main() {
	// 读取搜索结果
	resultData, _ := os.ReadFile("../data/dmhy_results.json")
	var results []SearchResult
	json.Unmarshal(resultData, &results)

	// 读取问题列表
	issueData, _ := os.ReadFile("../data/verify_issues.json")
	var issues []VerifyIssue
	json.Unmarshal(issueData, &issues)

	// 建立问题ID集合
	issueIDs := make(map[int]bool)
	for _, i := range issues {
		issueIDs[i.AnimeID] = true
	}

	// 筛选匹配成功的
	var output string
	count := 0
	for _, r := range results {
		if !issueIDs[r.AnimeID] && r.Magnet != "" {
			output += fmt.Sprintf("# %s (%d)\n%s\n\n", r.AnimeName, r.Year, r.Magnet)
			count++
		}
	}

	os.WriteFile("../data/matched_magnets.txt", []byte(output), 0644)
	fmt.Printf("已导出 %d 条匹配成功的磁链到 data/matched_magnets.txt\n", count)
}
