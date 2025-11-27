package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Config struct {
	Port        string `json:"port"`
	OpenListURL string `json:"openlist_url"`
}

var config Config

// OpenList API 响应结构
type OpenListResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type FileInfo struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	IsDir    bool   `json:"is_dir"`
	Modified string `json:"modified"`
	RawURL   string `json:"raw_url"`
	Thumb    string `json:"thumb"`
	Type     int    `json:"type"`
}

type ListData struct {
	Content []FileInfo `json:"content"`
	Total   int        `json:"total"`
}

func main() {
	loadConfig()

	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/api/list", handleList)
	http.HandleFunc("/api/get", handleGet)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	addr := ":" + config.Port
	log.Printf("动漫站启动在 http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func loadConfig() {
	config = Config{
		Port:        "8888",
		OpenListURL: "https://www.openlists.online",
	}

	data, err := os.ReadFile("config.json")
	if err == nil {
		json.Unmarshal(data, &config)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "static/index.html")
}

// 列出目录
func handleList(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	body := map[string]interface{}{
		"path":     path,
		"password": "",
		"page":     1,
		"per_page": 0,
		"refresh":  false,
	}

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

	body := map[string]interface{}{
		"path":     path,
		"password": "",
	}

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

	req, err := http.NewRequest("POST", config.OpenListURL+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 OpenList 失败: %v", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
