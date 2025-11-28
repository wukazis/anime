package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// 需要设置代理环境变量
func init() {
	os.Setenv("HTTP_PROXY", "http://127.0.0.1:10101")
	os.Setenv("HTTPS_PROXY", "http://127.0.0.1:10101")
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

// 映射记录
type AnimeMapping struct {
	AnimeName  string `json:"anime_name"`  // 番剧名（来自 matched_magnets.txt）
	FolderName string `json:"folder_name"` // PikPak 上的文件夹名
	FolderPath string `json:"folder_path"` // PikPak 上的完整路径
	FileID     string `json:"file_id"`     // PikPak 文件 ID
}

const (
	remoteName = "pikpak"
)

var (
	logFile     *os.File
	mappingFile *os.File
	mappings    []AnimeMapping
)

func main() {
	// 初始化日志文件
	var err error
	logFile, err = os.Create("../data/download_log.txt")
	if err != nil {
		fmt.Println("无法创建日志文件:", err)
		return
	}
	defer logFile.Close()

	// 从空映射开始（会覆盖旧文件）
	mappings = []AnimeMapping{}
	log("开始新的映射")

	tasks := loadTasks("../data/matched_magnets.txt")
	log(fmt.Sprintf("共加载 %d 个下载任务", len(tasks)))

	var failed []FailedTask
	var success int

	for i, task := range tasks {
		msg := fmt.Sprintf("[%d/%d] 添加离线任务: %s", i+1, len(tasks), task.Name)
		log(msg)

		folderName, fileID, err := addOfflineWithRclone(task)
		if err != nil {
			failed = append(failed, FailedTask{
				Name:   task.Name,
				Magnet: task.Magnet,
				Reason: err.Error(),
			})
			log(fmt.Sprintf("  ❌ 失败: %v", err))
		} else {
			success++
			// 添加映射
			mapping := AnimeMapping{
				AnimeName:  task.Name,
				FolderName: folderName,
				FolderPath: "wukazi/" + folderName,
				FileID:     fileID,
			}
			mappings = append(mappings, mapping)
			log(fmt.Sprintf("  ✅ 成功: %s -> %s", task.Name, folderName))

			// 每次成功后保存映射（防止中断丢失）
			saveMappings(mappings, "../data/anime_mapping.json")
		}

		time.Sleep(1 * time.Second)
	}

	saveFailedTasks(failed)
	log(fmt.Sprintf("\n完成！成功: %d, 失败: %d", success, len(failed)))
}


func log(msg string) {
	fmt.Println(msg)
	logFile.WriteString(time.Now().Format("2006-01-02 15:04:05") + " " + msg + "\n")
}

func addOfflineWithRclone(task Task) (folderName string, fileID string, err error) {
	magnet := task.Magnet
	if idx := strings.Index(magnet, "&tr="); idx > 0 {
		magnet = magnet[:idx]
	}

	cmd := exec.Command("rclone", "backend", "addurl", remoteName+":wukazi", magnet)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// 检查常见错误
	if strings.Contains(outputStr, "file_space_not_enough") {
		return "", "", fmt.Errorf("云盘空间不足")
	}
	if strings.Contains(outputStr, "task_daily_create_limit") {
		return "", "", fmt.Errorf("今日离线任务数已达上限")
	}

	if err != nil {
		return "", "", fmt.Errorf("%v: %s", err, outputStr)
	}

	// 解析返回的 JSON 获取文件信息
	// rclone addurl 返回格式: {"id":"xxx","file_name":"xxx",...}
	var result struct {
		ID       string `json:"id"`
		FileName string `json:"file_name"`
	}
	if err := json.Unmarshal(output, &result); err == nil && result.FileName != "" {
		return result.FileName, result.ID, nil
	}

	// 如果无法解析，返回空但不报错
	return "unknown", "", nil
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

func loadExistingMappings(filename string) []AnimeMapping {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	var mappings []AnimeMapping
	json.Unmarshal(data, &mappings)
	return mappings
}

func saveMappings(mappings []AnimeMapping, filename string) {
	data, _ := json.MarshalIndent(mappings, "", "  ")
	os.WriteFile(filename, data, 0644)
}

func saveFailedTasks(failed []FailedTask) {
	if len(failed) == 0 {
		log("没有失败的任务")
		return
	}

	file, err := os.Create("../data/rclone_failed.txt")
	if err != nil {
		log("无法创建失败列表文件: " + err.Error())
		return
	}
	defer file.Close()

	for _, f := range failed {
		file.WriteString(fmt.Sprintf("# %s\n# 原因: %s\n%s\n\n", f.Name, f.Reason, f.Magnet))
	}
	log("失败列表已保存到 data/rclone_failed.txt")
}
