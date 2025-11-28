package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// PikPak 配置
type Config struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	AccessToken string `json:"access_token"`
	FolderID    string `json:"folder_id"`
}

type Task struct {
	Name   string
	Magnet string
}

type FailedTask struct {
	Name   string
	Magnet string
	Reason string
}

// Android 客户端参数
const (
	ClientID      = "YNxT9w7GMdWvEOKa"
	ClientSecret  = "dbw2OtmVEeuUvIptb1Coyg"
	ClientVersion = "1.53.2"
	PackageName   = "com.pikcloud.pikpak"
	SdkVersion    = "2.0.6.206003"
)

// 签名算法
var Algorithms = []string{
	"SOP04dGzk0TNO7t7t9ekDbAmx+eq0OI1ovEx",
	"nVBjhYiND4hZ2NCGyV5beamIr7k6ifAsAbl",
	"Ddjpt5B/Cit6EDq2a6cXgxY9lkEIOw4yC1GDF28KrA",
	"VVCogcmSNIVvgV6U+AochorydiSymi68YVNGiz",
	"u5ujk5sM62gpJOsB/1Gu/zsfgfZO",
	"dXYIiBOAHZgzSruaQ2Nhrqc2im",
	"z5jUTBSIpBN9g4qSJGlidNAutX6",
	"KJE2oveZ34du/g1tiimm",
}

var config Config
var accessToken string
var captchaToken string
var deviceID string
var client *http.Client

func init() {
	// 设置代理
	proxyURL, _ := url.Parse("http://127.0.0.1:10101")
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client = &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
}

func main() {
	loadConfig()
	deviceID = generateDeviceID()

	// 获取 captcha token
	if err := refreshCaptchaToken("POST:/drive/v1/files"); err != nil {
		fmt.Println("获取 captcha token 失败:", err)
		return
	}
	fmt.Println("✅ 获取 captcha token 成功")

	if config.AccessToken != "" {
		accessToken = config.AccessToken
		fmt.Println("✅ 使用配置的 access_token")
	} else {
		if err := login(); err != nil {
			fmt.Println("登录失败:", err)
			return
		}
		fmt.Println("✅ 登录成功")
	}

	tasks := loadTasks("../data/matched_magnets.txt")
	fmt.Printf("共加载 %d 个下载任务\n", len(tasks))

	var failed []FailedTask
	var success int

	for i, task := range tasks {
		fmt.Printf("[%d/%d] 添加离线任务: %s\n", i+1, len(tasks), task.Name)

		err := addOfflineTask(task)
		if err != nil {
			failed = append(failed, FailedTask{
				Name:   task.Name,
				Magnet: task.Magnet,
				Reason: err.Error(),
			})
			fmt.Printf("  ❌ 失败: %v\n", err)
		} else {
			success++
			fmt.Printf("  ✅ 成功\n")
		}

		time.Sleep(500 * time.Millisecond)
	}

	saveFailedTasks(failed)
	fmt.Printf("\n完成！成功: %d, 失败: %d\n", success, len(failed))
}

func loadConfig() {
	data, err := os.ReadFile("pikpak_config.json")
	if err != nil {
		fmt.Println("请创建 pikpak_config.json 配置文件")
		os.Exit(1)
	}
	json.Unmarshal(data, &config)
}

func generateDeviceID() string {
	// 生成一个固定的设备ID
	h := md5.New()
	h.Write([]byte(config.Username + "pikpak_device"))
	return hex.EncodeToString(h.Sum(nil))
}

func getMD5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func getCaptchaSign() (string, string) {
	timestamp := fmt.Sprint(time.Now().UnixMilli())
	str := ClientID + ClientVersion + PackageName + deviceID + timestamp
	for _, alg := range Algorithms {
		str = getMD5(str + alg)
	}
	return timestamp, "1." + str
}

func refreshCaptchaToken(action string) error {
	timestamp, captchaSign := getCaptchaSign()

	payload := map[string]interface{}{
		"action":        action,
		"captcha_token": captchaToken,
		"client_id":     ClientID,
		"device_id":     deviceID,
		"meta": map[string]string{
			"client_version": ClientVersion,
			"package_name":   PackageName,
			"user_id":        "",
			"timestamp":      timestamp,
			"captcha_sign":   captchaSign,
		},
	}

	jsonBody, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://user.mypikpak.com/v1/shield/captcha/init", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-ID", ClientID)
	req.Header.Set("X-Device-ID", deviceID)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if token, ok := result["captcha_token"].(string); ok {
		captchaToken = token
		return nil
	}

	return fmt.Errorf("获取 captcha token 失败: %s", string(body))
}

func login() error {
	payload := map[string]string{
		"client_id":     ClientID,
		"client_secret": ClientSecret,
		"username":      config.Username,
		"password":      config.Password,
	}

	jsonBody, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://user.mypikpak.com/v1/auth/signin", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-ID", ClientID)
	req.Header.Set("X-Device-ID", deviceID)
	req.Header.Set("X-Captcha-Token", captchaToken)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if token, ok := result["access_token"].(string); ok {
		accessToken = token
		return nil
	}

	return fmt.Errorf("登录响应: %s", string(body))
}

func addOfflineTask(task Task) error {
	// 刷新 captcha token
	fmt.Println("  [DEBUG] 刷新 captcha token...")
	if err := refreshCaptchaToken("POST:/drive/v1/files"); err != nil {
		fmt.Println("  [DEBUG] 刷新 captcha 失败:", err)
		return err
	}
	fmt.Println("  [DEBUG] captcha token 获取成功")

	payload := map[string]interface{}{
		"kind":        "drive#file",
		"upload_type": "UPLOAD_TYPE_URL",
		"url": map[string]string{
			"url": task.Magnet,
		},
		"parent_id":   config.FolderID,
		"folder_type": "",
	}

	jsonBody, _ := json.Marshal(payload)
	fmt.Println("  [DEBUG] 发送请求到 PikPak API...")
	fmt.Printf("  [DEBUG] 磁链长度: %d 字符\n", len(task.Magnet))
	
	req, _ := http.NewRequest("POST", "https://api-drive.mypikpak.com/drive/v1/files", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Client-ID", ClientID)
	req.Header.Set("X-Device-ID", deviceID)
	req.Header.Set("X-Captcha-Token", captchaToken)

	fmt.Println("  [DEBUG] 等待响应...")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("  [DEBUG] 请求失败:", err)
		return err
	}
	defer resp.Body.Close()
	fmt.Printf("  [DEBUG] 响应状态码: %d\n", resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("  [DEBUG] 响应内容: %s\n", string(body)[:min(200, len(body))])
	
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// 检查错误码 9 需要刷新 captcha
	if errCode, ok := result["error_code"].(float64); ok && int(errCode) == 9 {
		fmt.Println("  [DEBUG] 需要刷新 captcha，重试...")
		if err := refreshCaptchaToken("POST:/drive/v1/files"); err != nil {
			return err
		}
		return addOfflineTask(task) // 重试
	}

	if _, ok := result["file"]; ok {
		return nil
	}
	if _, ok := result["task"]; ok {
		return nil
	}
	if errMsg, ok := result["error"].(string); ok {
		return fmt.Errorf(errMsg)
	}

	return fmt.Errorf("未知响应: %s", string(body))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func loadTasks(filename string) []Task {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("无法打开文件:", err)
		return nil
	}
	defer file.Close()

	var tasks []Task
	var currentName string
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			currentName = strings.TrimPrefix(line, "# ")
		} else if strings.HasPrefix(line, "magnet:") && currentName != "" {
			tasks = append(tasks, Task{Name: currentName, Magnet: line})
		}
	}
	return tasks
}

func saveFailedTasks(failed []FailedTask) {
	if len(failed) == 0 {
		fmt.Println("没有失败的任务")
		return
	}

	file, err := os.Create("../data/pikpak_failed.txt")
	if err != nil {
		fmt.Println("无法创建失败列表文件:", err)
		return
	}
	defer file.Close()

	for _, f := range failed {
		file.WriteString(fmt.Sprintf("# %s\n# 原因: %s\n%s\n\n", f.Name, f.Reason, f.Magnet))
	}
	fmt.Printf("失败列表已保存到 data/pikpak_failed.txt\n")
}
