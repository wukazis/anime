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
	Magnet    string `json:"magnet"`
}

type VerifyIssue struct {
	AnimeID int `json:"anime_id"`
}

func main() {
	resultData, _ := os.ReadFile("../data/dmhy_results.json")
	var results []SearchResult
	json.Unmarshal(resultData, &results)

	issueData, _ := os.ReadFile("../data/verify_issues.json")
	var issues []VerifyIssue
	json.Unmarshal(issueData, &issues)

	issueIDs := make(map[int]bool)
	for _, i := range issues {
		issueIDs[i.AnimeID] = true
	}

	emptyCount := 0
	for _, r := range results {
		if !issueIDs[r.AnimeID] && r.Magnet == "" {
			fmt.Printf("无磁链: %s (%d)\n", r.AnimeName, r.Year)
			emptyCount++
		}
	}
	fmt.Printf("\n共 %d 条匹配成功但无磁链\n", emptyCount)
}
