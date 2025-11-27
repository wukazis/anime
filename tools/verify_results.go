package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type AnimeInfo struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	NameCN string `json:"name_cn"`
	Year   int    `json:"year"`
}

type SearchResult struct {
	AnimeName string `json:"anime_name"`
	AnimeID   int    `json:"anime_id"`
	Year      int    `json:"year"`
	Title     string `json:"title"`
	Magnet    string `json:"magnet"`
	Score     int    `json:"score"`
}

type VerifyResult struct {
	AnimeName   string `json:"anime_name"`
	AnimeID     int    `json:"anime_id"`
	Year        int    `json:"year"`
	SearchTitle string `json:"search_title"`
	Status      string `json:"status"` // ok, mismatch, season_mismatch, not_found
	Issue       string `json:"issue"`
}

func main() {
	// 读取番剧数据
	animeData, _ := os.ReadFile("../data/anime_db.json")
	var animes []AnimeInfo
	json.Unmarshal(animeData, &animes)

	// 读取搜索结果
	resultData, _ := os.ReadFile("../data/dmhy_results.json")
	var results []SearchResult
	json.Unmarshal(resultData, &results)

	// 建立映射
	resultMap := make(map[int]SearchResult)
	for _, r := range results {
		resultMap[r.AnimeID] = r
	}

	var verified []VerifyResult
	var issues []VerifyResult

	for _, anime := range animes {
		name := anime.NameCN
		if name == "" {
			name = anime.Name
		}

		result, found := resultMap[anime.ID]
		if !found {
			v := VerifyResult{
				AnimeName: name,
				AnimeID:   anime.ID,
				Year:      anime.Year,
				Status:    "not_found",
				Issue:     "未找到资源",
			}
			issues = append(issues, v)
			continue
		}

		v := VerifyResult{
			AnimeName:   name,
			AnimeID:     anime.ID,
			Year:        anime.Year,
			SearchTitle: result.Title,
			Status:      "ok",
		}

		// 检查季数匹配
		animeSeason := extractSeason(name)
		titleSeason := extractSeason(result.Title)

		if animeSeason != titleSeason {
			if animeSeason == 0 && titleSeason > 1 {
				v.Status = "season_mismatch"
				v.Issue = fmt.Sprintf("番剧无季数标记(默认第1季)，但资源是第%d季", titleSeason)
				issues = append(issues, v)
				continue
			}
			if animeSeason > 0 && titleSeason != animeSeason {
				v.Status = "season_mismatch"
				v.Issue = fmt.Sprintf("番剧第%d季，资源第%d季", animeSeason, titleSeason)
				issues = append(issues, v)
				continue
			}
		}

		// 检查名称相似度
		if !isNameMatch(name, result.Title) {
			v.Status = "mismatch"
			v.Issue = "名称可能不匹配"
			issues = append(issues, v)
			continue
		}

		verified = append(verified, v)
	}

	// 保存结果
	saveVerifyResults(verified, issues)
}

func extractSeason(s string) int {
	s = strings.ToLower(s)
	
	// 匹配各种季数表示
	patterns := []struct {
		re     *regexp.Regexp
		season int
	}{
		{regexp.MustCompile(`第([一二三四五六七八九十\d]+)季`), 0},
		{regexp.MustCompile(`season\s*(\d+)`), 0},
		{regexp.MustCompile(`s(\d+)`), 0},
		{regexp.MustCompile(`(\d+)(?:nd|rd|th)\s*season`), 0},
		{regexp.MustCompile(`ii+`), 0}, // II, III 等
	}

	// 中文数字转换
	cnNum := map[string]int{
		"一": 1, "二": 2, "三": 3, "四": 4, "五": 5,
		"六": 6, "七": 7, "八": 8, "九": 9, "十": 10,
	}

	// 检查 "第X季"
	re := regexp.MustCompile(`第([一二三四五六七八九十\d]+)季`)
	if match := re.FindStringSubmatch(s); len(match) > 1 {
		if n, ok := cnNum[match[1]]; ok {
			return n
		}
		var n int
		fmt.Sscanf(match[1], "%d", &n)
		return n
	}

	// 检查 Season X
	re = regexp.MustCompile(`season\s*(\d+)`)
	if match := re.FindStringSubmatch(s); len(match) > 1 {
		var n int
		fmt.Sscanf(match[1], "%d", &n)
		return n
	}

	// 检查 SX
	re = regexp.MustCompile(`\bs(\d+)\b`)
	if match := re.FindStringSubmatch(s); len(match) > 1 {
		var n int
		fmt.Sscanf(match[1], "%d", &n)
		return n
	}

	// 检查 II, III
	if strings.Contains(s, "iii") {
		return 3
	}
	if strings.Contains(s, "ii") && !strings.Contains(s, "iii") {
		return 2
	}

	// 检查标题末尾数字 如 "进击的巨人2"
	re = regexp.MustCompile(`(\d+)$`)
	if match := re.FindStringSubmatch(strings.TrimSpace(s)); len(match) > 1 {
		var n int
		fmt.Sscanf(match[1], "%d", &n)
		if n >= 2 && n <= 10 {
			return n
		}
	}

	return 0 // 默认第1季或无季数
}

func isNameMatch(animeName, title string) bool {
	animeName = strings.ToLower(animeName)
	title = strings.ToLower(title)

	// 移除常见干扰词
	cleanName := regexp.MustCompile(`[第一二三四五六七八九十\d]+季`).ReplaceAllString(animeName, "")
	cleanName = strings.TrimSpace(cleanName)

	// 检查番剧名是否在标题中
	if strings.Contains(title, cleanName) {
		return true
	}

	// 检查主要关键词
	words := strings.Fields(cleanName)
	matchCount := 0
	for _, w := range words {
		if len(w) > 1 && strings.Contains(title, w) {
			matchCount++
		}
	}

	return matchCount >= len(words)/2
}

func saveVerifyResults(verified, issues []VerifyResult) {
	// 保存问题列表
	issueData, _ := json.MarshalIndent(issues, "", "  ")
	os.WriteFile("../data/verify_issues.json", issueData, 0644)

	// 生成可读报告
	var report string
	report += fmt.Sprintf("=== 校对报告 ===\n")
	report += fmt.Sprintf("匹配成功: %d\n", len(verified))
	report += fmt.Sprintf("存在问题: %d\n\n", len(issues))

	if len(issues) > 0 {
		report += "=== 问题列表 ===\n\n"
		for _, v := range issues {
			report += fmt.Sprintf("【%s】(%d年)\n", v.AnimeName, v.Year)
			report += fmt.Sprintf("  状态: %s\n", v.Status)
			report += fmt.Sprintf("  问题: %s\n", v.Issue)
			if v.SearchTitle != "" {
				report += fmt.Sprintf("  搜索结果: %s\n", truncate(v.SearchTitle, 60))
			}
			report += "\n"
		}
	}

	os.WriteFile("../data/verify_report.txt", []byte(report), 0644)

	fmt.Print(report)
	fmt.Println("\n结果已保存到:")
	fmt.Println("  - data/verify_issues.json")
	fmt.Println("  - data/verify_report.txt")
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "..."
	}
	return s
}
