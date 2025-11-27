package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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

type UploadResponse struct {
	Success  bool   `json:"success"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Error    string `json:"error"`
}

const (
	imageBedURL = "http://204.44.122.166:8090/api/upload"
	tempDir     = "./covers_temp"
)

var client = &http.Client{Timeout: 60 * time.Second}

func main() {
	os.MkdirAll(tempDir, 0755)

	// 读取番剧数据
	data, err := os.ReadFile("../data/anime_db.json")
	if err != nil {
		fmt.Println("读取 anime_db.json 失败:", err)
		return
	}

	var animes []AnimeInfo
	json.Unmarshal(data, &animes)
	fmt.Printf("共 %d 部番剧\n", len(animes))

	updated := 0
	failed := 0

	for i, anime := range animes {
		if anime.Cover == "" || isMyImageBed(anime.Cover) {
			continue
		}

		fmt.Printf("[%d/%d] %s ... ", i+1, len(animes), anime.NameCN)

		newURL, err := downloadAndUpload(anime.Cover, anime.ID)
		if err != nil {
			fmt.Printf("失败: %v\n", err)
			failed++
			continue
		}

		animes[i].Cover = newURL
		updated++
		fmt.Printf("成功\n")

		time.Sleep(500 * time.Millisecond) // 避免请求过快
	}

	// 保存更新后的数据
	newData, _ := json.MarshalIndent(animes, "", "  ")
	os.WriteFile("../data/anime_db.json", newData, 0644)

	fmt.Printf("\n完成! 更新 %d 张, 失败 %d 张\n", updated, failed)
}

func isMyImageBed(url string) bool {
	return len(url) > 0 && (url[:4] != "http" || 
		bytes.Contains([]byte(url), []byte("204.44.122.166:8090")))
}

func downloadAndUpload(coverURL string, animeID int) (string, error) {
	// 下载图片
	resp, err := client.Get(coverURL)
	if err != nil {
		return "", fmt.Errorf("下载失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("下载返回 %d", resp.StatusCode)
	}

	// 获取扩展名
	ext := filepath.Ext(coverURL)
	if ext == "" || len(ext) > 5 {
		ext = ".jpg"
	}

	// 保存到临时文件
	tempFile := filepath.Join(tempDir, fmt.Sprintf("%d%s", animeID, ext))
	imgData, _ := io.ReadAll(resp.Body)
	os.WriteFile(tempFile, imgData, 0644)
	defer os.Remove(tempFile)

	// 上传到图床
	return uploadToImageBed(tempFile)
}

func uploadToImageBed(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 创建 multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filepath.Base(filePath))
	io.Copy(part, file)
	writer.Close()

	req, _ := http.NewRequest("POST", imageBedURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("上传失败: %v", err)
	}
	defer resp.Body.Close()

	var result UploadResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success {
		return "", fmt.Errorf("上传失败: %s", result.Error)
	}

	return result.URL, nil
}
