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
	OpenListURL string `json:"openlist_url"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	SavePath    string `json:"save_path"` // PikPak 保存路径，如 /pikpak/动漫
}

type LoginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token string `json:"token"`
	} `json:"data"`
}

type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var (
	config Config
	client = &http.Client{Timeout: 60 * time.Second}
	token  string
)

func main() {
	loadConfig()

	// 登录获取 token
	fmt.Println("登录 OpenList...")
	if err := login(); err != nil {
		fmt.Printf("登录失败: %v\n", err)
		return
	}
	fmt.Println("登录成功")

	// 读取磁力链接
	magnets := loadMagnets()
	fmt.Printf("共 %d 个磁力链接\n\n", len(magnets))

	success := 0
	failed := 0

	for i, m := range magnets {
		fmt.Printf("[%d/%d] %s ... ", i+1, len(magnets), m.name)

		err := addOfflineTask(m.magnet, m.name)
		if err != nil {
			fmt.Printf("失败: %v\n", err)
			failed++
		} else {
			fmt.Println("成功")
			success++
		}

		time.Sleep(2 * time.Second) // 避免请求过快
	}

	fmt.Printf("\n完成! 成功 %d, 失败 %d\n", success, failed)
}

func loadConfig() {
	config = Config{
		OpenListURL: "https://www.openlists.online",
		Username:    "admin",
		Password:    "",
		SavePath:    "/pikpak/动漫",
	}

	data, err := os.ReadFile("offline_config.json")
	if err != nil {
		fmt.Println("请创建 offline_config.json 配置文件:")
		fmt.Println(`{
  "openlist_url": "https://www.openlists.online",
  "username": "admin",
  "password": "你的密码",
  "save_path": "/pikpak/动漫"
}`)
		os.Exit(1)
	}
	json.Unmarshal(data, &config)
}

func login() error {
	body := map[string]string{
		"username": config.Username,
		"password": config.Password,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := client.Post(config.OpenListURL+"/api/auth/login", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result LoginResponse
	json.Unmarshal(data, &result)

	if result.Code != 200 {
		return fmt.Errorf("%s", result.Message)
	}

	token = result.Data.Token
	return nil
}

type magnetInfo struct {
	name   string
	magnet string
}

func loadMagnets() []magnetInfo {
	var magnets []magnetInfo

	file, err := os.Open("../data/matched_magnets.txt")
	if err != nil {
		fmt.Println("无法读取 matched_magnets.txt")
		return magnets
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentName string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			currentName = strings.TrimPrefix(line, "# ")
		} else if strings.HasPrefix(line, "magnet:") {
			magnets = append(magnets, magnetInfo{
				name:   currentName,
				magnet: line,
			})
		}
	}

	return magnets
}

func addOfflineTask(magnet, name string) error {
	// 创建以番剧名命名的文件夹
	savePath := config.SavePath
	if name != "" {
		// 提取番剧名（去掉年份）
		parts := strings.Split(name, " (")
		if len(parts) > 0 {
			savePath = config.SavePath + "/" + parts[0]
		}
	}

	body := map[string]interface{}{
		"path": savePath,
		"urls": []string{magnet},
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", config.OpenListURL+"/api/fs/add_offline_download", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result APIResponse
	json.Unmarshal(data, &result)

	if result.Code != 200 {
		return fmt.Errorf("%s", result.Message)
	}

	return nil
}
