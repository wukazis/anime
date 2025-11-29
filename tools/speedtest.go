package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	rcloneURL   = "http://3.149.250.103:5555"
	openlistURL = "http://3.149.250.103:5244"
	testFile    = "/anime/[Airota][AIR][BDRip 1080p HEVC-yuv444p10 FLAC]/[Airota][AIR][01][BDRip 1080p HEVC-yuv444p10 FLAC].mkv"
	testSize    = 1 * 1024 * 1024 // 下载 1MB 测速
)

func main() {
	fmt.Println("=== 直链速度测试 ===\n")

	// 测试 rclone serve
	fmt.Println("1. 测试 Rclone Serve 直链...")
	rcloneSpeed := testRclone()

	// 测试 OpenList
	fmt.Println("\n2. 测试 OpenList 直链...")
	openlistSpeed := testOpenList()

	// 结果对比
	fmt.Println("\n=== 测试结果 ===")
	fmt.Printf("Rclone Serve: %.2f MB/s\n", rcloneSpeed)
	fmt.Printf("OpenList:     %.2f MB/s\n", openlistSpeed)

	if rcloneSpeed > openlistSpeed {
		fmt.Printf("\nRclone 快 %.1f%%\n", (rcloneSpeed-openlistSpeed)/openlistSpeed*100)
	} else {
		fmt.Printf("\nOpenList 快 %.1f%%\n", (openlistSpeed-rcloneSpeed)/rcloneSpeed*100)
	}
}

func testRclone() float64 {
	url := rcloneURL + testFile
	return downloadSpeed(url, "Rclone")
}

func testOpenList() float64 {
	// 先获取直链
	body := map[string]interface{}{"path": "/onedrive" + testFile, "password": ""}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(openlistURL+"/api/fs/get", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("  获取直链失败: %v\n", err)
		return 0
	}
	defer resp.Body.Close()

	var result struct {
		Code int `json:"code"`
		Data struct {
			RawURL string `json:"raw_url"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Code != 200 || result.Data.RawURL == "" {
		fmt.Println("  获取直链失败")
		return 0
	}

	return downloadSpeed(result.Data.RawURL, "OpenList")
}

func downloadSpeed(url, name string) float64 {
	client := &http.Client{Timeout: 60 * time.Second}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", testSize-1))

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
		return 0
	}
	defer resp.Body.Close()

	n, _ := io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(start).Seconds()

	speed := float64(n) / elapsed / 1024 / 1024
	fmt.Printf("  下载 %d bytes, 耗时 %.2fs, 速度 %.2f MB/s\n", n, elapsed, speed)
	return speed
}
