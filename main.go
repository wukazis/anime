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

type AnimeMapping struct {
	AnimeName  string   `json:"anime_name"`
	FolderName string   `json:"folder_name"`
	FolderPath string   `json:"folder_path"`
	Episodes   []string `json:"episodes"`
}

var config Config
var animeDB []AnimeInfo
var animeMapping map[string]bool            // 番剧名 -> 是否有资源
var animeFolderPath map[string]string       // 番剧名 -> 文件夹路径
var animeEpisodes map[string][]string       // 番剧名 -> 视频文件列表

func main() {
	loadConfig()
	loadAnimeDB()
	loadAnimeMapping()

	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/api/anime", handleAnimeList)
	http.HandleFunc("/api/anime/search", handleAnimeSearch)
	http.HandleFunc("/api/anime/episodes", handleAnimeEpisodes)
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
	animeFolderPath = make(map[string]string)
	animeEpisodes = make(map[string][]string)
	data, err := os.ReadFile("data/anime_mapping_onedrive.json")
	if err != nil {
		log.Printf("警告: 无法加载映射表: %v", err)
		return
	}
	var mappings []AnimeMapping
	json.Unmarshal(data, &mappings)
	for _, m := range mappings {
		animeMapping[m.AnimeName] = true
		animeFolderPath[m.AnimeName] = m.FolderPath
		animeEpisodes[m.AnimeName] = m.Episodes
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

// 获取番剧的视频文件列表
func handleAnimeEpisodes(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	year := r.URL.Query().Get("year")
	if name == "" || year == "" {
		http.Error(w, "name and year required", 400)
		return
	}

	key := fmt.Sprintf("%s (%s)", name, year)
	folderPath, ok := animeFolderPath[key]
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"episodes": []interface{}{}})
		return
	}

	// 转换路径格式：onedrive:anime/xxx -> /onedrive/anime/xxx
	var apiPath string
	if strings.HasPrefix(folderPath, "onedrive:") {
		apiPath = "/" + strings.Replace(folderPath, ":", "/", 1)
	} else {
		apiPath = "/pikpak/" + folderPath
	}

	// 从映射表读取 episodes
	eps := animeEpisodes[key]
	var episodes []map[string]string
	for _, epName := range eps {
		episodes = append(episodes, map[string]string{
			"name": epName,
			"path": apiPath + "/" + epName,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"folder_path": apiPath,
		"episodes":    episodes,
	})
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
	baseURL := strings.TrimSuffix(config.OpenListURL, "/")
	req, _ := http.NewRequest("POST", baseURL+endpoint, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 OpenList 失败: %v", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
