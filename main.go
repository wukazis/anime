package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port        string `json:"port"`
	OpenListURL string `json:"openlist_url"`
}

type AnimeInfo struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	NameCN      string   `json:"name_cn"`
	Year        int      `json:"year"`
	Date        string   `json:"date"`
	Summary     string   `json:"summary"`
	Cover       string   `json:"cover"`
	Score       float64  `json:"score"`
	Tags        []string `json:"tags"`
	HasResource bool     `json:"has_resource"`
}

var config Config
var animeDB []AnimeInfo
var animeMapping map[string]bool // 番剧名 -> 是否有资源

func main() {
	loadConfig()
	loadAnimeDB()
	loadAnimeMapping()

	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/api/anime", handleAnimeList)
	http.HandleFunc("/api/anime/search", handleAnimeSearch)
	http.HandleFunc("/api/list", handleList)
	http.HandleFunc("/api/get", handleGet)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	addr := ":" + config.Port
	log.Printf("动漫站启动在 http://localhost%s", addr)
	log.Printf("已加载 %d 部番剧数据", len(animeDB))
	log.Fatal(http.ListenAndServe(addr, nil))
}

func loadConfig() {
	config = Config{
		Port:        "8888",
		OpenListURL: "https://www.openlists.online",
	}
	data, _ := os.ReadFile("config.json")
	json.Unmarshal(data, &config)
}

func loadAnimeDB() {
	data, err := os.ReadFile("data/anime_db.json")
	if err != nil {
		log.Printf("警告: 无法加载番剧数据库: %v", err)
		return
	}
	json.Unmarshal(data, &animeDB)
}

func loadAnimeMapping() {
	animeMapping = make(map[string]bool)
	data, err := os.ReadFile("data/anime_mapping.json")
	if err != nil {
		log.Printf("警告: 无法加载映射表: %v", err)
		return
	}
	var mappings []struct {
		AnimeName string `json:"anime_name"`
	}
	json.Unmarshal(data, &mappings)
	for _, m := range mappings {
		animeMapping[m.AnimeName] = true
	}
	log.Printf("已加载 %d 条资源映射", len(animeMapping))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "static/index.html")
}

// 获取番剧列表（支持年份筛选和分页）
func handleAnimeList(w http.ResponseWriter, r *http.Request) {
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize := 24
	if page < 1 { page = 1 }

	var filtered []AnimeInfo
	for _, a := range animeDB {
		if year == 0 || a.Year == year {
			// 检查是否有资源
			key := fmt.Sprintf("%s (%d)", a.NameCN, a.Year)
			if a.NameCN == "" {
				key = fmt.Sprintf("%s (%d)", a.Name, a.Year)
			}
			a.HasResource = animeMapping[key]
			filtered = append(filtered, a)
		}
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total { start = total }
	if end > total { end = total }

	result := map[string]interface{}{
		"total": total,
		"page":  page,
		"data":  filtered[start:end],
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// 搜索番剧
func handleAnimeSearch(w http.ResponseWriter, r *http.Request) {
	keyword := strings.ToLower(r.URL.Query().Get("q"))
	if keyword == "" {
		json.NewEncoder(w).Encode([]AnimeInfo{})
		return
	}

	var results []AnimeInfo
	for _, a := range animeDB {
		if strings.Contains(strings.ToLower(a.Name), keyword) ||
			strings.Contains(strings.ToLower(a.NameCN), keyword) {
			key := fmt.Sprintf("%s (%d)", a.NameCN, a.Year)
			if a.NameCN == "" {
				key = fmt.Sprintf("%s (%d)", a.Name, a.Year)
			}
			a.HasResource = animeMapping[key]
			results = append(results, a)
			if len(results) >= 50 { break }
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// OpenList 目录列表
func handleList(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" { path = "/" }

	body := map[string]interface{}{"path": path, "password": "", "page": 1, "per_page": 0, "refresh": false}
	resp, err := callOpenList("/api/fs/list", body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

// 获取文件直链
func handleGet(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", 400)
		return
	}

	body := map[string]interface{}{"path": path, "password": ""}
	resp, err := callOpenList("/api/fs/get", body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func callOpenList(endpoint string, body map[string]interface{}) ([]byte, error) {
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", config.OpenListURL+endpoint, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 OpenList 失败: %v", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
