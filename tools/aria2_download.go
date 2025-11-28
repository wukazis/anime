package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	Aria2URL    string `json:"aria2_url"`
	Aria2Secret string `json:"aria2_secret"`
	DownloadDir string `json:"download_dir"`
}

type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	ID      string        `json:"id"`
	Params  []interface{} `json:"params"`
}

type RPCResponse struct {
	ID     string `json:"id"`
	Result string `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

var config Config
var client = &http.Client{Timeout: 30 * time.Second}

func main() {
	loadConfig()

	magnets := loadMagnets()
	fmt.Printf("共 %d 条磁力链接\n\n", len(magnets))

	success := 0
	failed := 0

	for i, m := range magnets {
		fmt.Printf("[%d/%d] %s ... ", i+1, len(magnets), m.name)

		err := addTask(m.magnet, m.name)
		if err != nil {
			fmt.Printf("失败: %v\n", err)
			failed++
		} else {
			fmt.Println("成功")
			success++
		}

		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("\n完成! 成功 %d, 失败 %d\n", success, failed)
}

func loadConfig() {
	config = Config{
		Aria2URL:    "http://127.0.0.1:6800/jsonrpc",
		Aria2Secret: "123456",
		DownloadDir: "/tmp/aria2/anime",
	}

	data, err := os.ReadFile("aria2_config.json")
	if err != nil {
		fmt.Println("请创建 aria2_config.json:")
		fmt.Println(`{
  "aria2_url": "http://127.0.0.1:6800/jsonrpc",
  "aria2_secret": "你的aria2密码",
  "download_dir": "/tmp/aria2/anime"
}`)
		os.Exit(1)
	}
	json.Unmarshal(data, &config)
}

type magnetInfo struct {
	name   string
	magnet string
}

func loadMagnets() []magnetInfo {
	var magnets []magnetInfo
	file, _ := os.Open("../data/matched_magnets.txt")
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentName string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			currentName = strings.TrimPrefix(line, "# ")
		} else if strings.HasPrefix(line, "magnet:") {
			magnets = append(magnets, magnetInfo{name: currentName, magnet: line})
		}
	}
	return magnets
}

func addTask(magnet, name string) error {
	// 提取番剧名作为子目录
	dir := config.DownloadDir
	if name != "" {
		parts := strings.Split(name, " (")
		if len(parts) > 0 {
			dir = config.DownloadDir + "/" + parts[0]
		}
	}

	req := RPCRequest{
		JSONRPC: "2.0",
		Method:  "aria2.addUri",
		ID:      "1",
		Params: []interface{}{
			"token:" + config.Aria2Secret,
			[]string{magnet},
			map[string]string{"dir": dir},
		},
	}

	jsonBody, _ := json.Marshal(req)
	resp, err := client.Post(config.Aria2URL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result RPCResponse
	json.Unmarshal(body, &result)

	if result.Error != nil {
		return fmt.Errorf("%s", result.Error.Message)
	}
	return nil
}
