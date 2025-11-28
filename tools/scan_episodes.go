package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type AnimeMapping struct {
	AnimeName  string   `json:"anime_name"`
	FolderName string   `json:"folder_name"`
	FolderPath string   `json:"folder_path"`
	FileID     string   `json:"file_id"`
	Episodes   []string `json:"episodes,omitempty"`
}

func main() {
	// 读取映射表
	data, err := os.ReadFile("../data/anime_mapping_onedrive.json")
	if err != nil {
		fmt.Println("读取映射表失败:", err)
		return
	}

	var mappings []AnimeMapping
	json.Unmarshal(data, &mappings)
	fmt.Printf("共 %d 条映射\n", len(mappings))

	// 扫描每个文件夹
	for i := range mappings {
		m := &mappings[i]
		fmt.Printf("[%d/%d] 扫描: %s\n", i+1, len(mappings), m.AnimeName)

		episodes := scanFolderRclone(m.FolderPath)
		m.Episodes = episodes
		fmt.Printf("  -> 找到 %d 个视频\n", len(episodes))
	}

	// 保存更新后的映射表
	output, _ := json.MarshalIndent(mappings, "", "  ")
	os.WriteFile("../data/anime_mapping_onedrive.json", output, 0644)
	fmt.Println("映射表已更新")
}

func scanFolderRclone(folderPath string) []string {
	// 转换路径: onedrive:anime/xxx -> onedrive:anime/xxx (rclone格式)
	// folderPath 已经是 onedrive:anime/xxx 格式
	rclonePath := folderPath

	// 使用 rclone lsf 列出文件
	cmd := exec.Command("rclone", "lsf", rclonePath, "--files-only")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("  rclone 失败: %v\n", err)
		return nil
	}

	// 解析文件列表
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var videos []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && isVideoFile(line) {
			videos = append(videos, line)
		}
	}

	// 排序
	sort.Strings(videos)
	return videos
}

func isVideoFile(name string) bool {
	name = strings.ToLower(name)
	return strings.HasSuffix(name, ".mp4") || strings.HasSuffix(name, ".mkv") ||
		strings.HasSuffix(name, ".avi") || strings.HasSuffix(name, ".webm") ||
		strings.HasSuffix(name, ".flv") || strings.HasSuffix(name, ".mov")
}
