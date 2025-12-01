package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
	// 1. 读取 pikpak 映射表
	pikpakData, err := os.ReadFile("../data/anime_mapping.json")
	if err != nil {
		fmt.Println("读取 pikpak 映射表失败:", err)
		return
	}

	var pikpakMappings []AnimeMapping
	json.Unmarshal(pikpakData, &pikpakMappings)
	fmt.Printf("PikPak 映射表共 %d 条\n", len(pikpakMappings))

	// 2. 获取 OneDrive 文件夹列表
	fmt.Println("正在扫描 OneDrive anime 文件夹...")
	onedriveFolders := getOneDriveFolders()
	fmt.Printf("OneDrive 共 %d 个文件夹\n", len(onedriveFolders))

	// 3. 匹配并生成新的 onedrive 映射表
	var onedriveMappings []AnimeMapping
	matched := 0

	for _, m := range pikpakMappings {
		// 跳过已经是 onedrive 路径的
		if strings.HasPrefix(m.FolderPath, "onedrive:") {
			onedriveMappings = append(onedriveMappings, m)
			matched++
			continue
		}

		// 在 OneDrive 中查找匹配的文件夹
		folderName := m.FolderName
		if found, path := findInOneDrive(folderName, onedriveFolders); found {
			newMapping := AnimeMapping{
				AnimeName:  m.AnimeName,
				FolderName: folderName,
				FolderPath: "onedrive:anime/" + path,
				FileID:     m.FileID,
			}
			onedriveMappings = append(onedriveMappings, newMapping)
			matched++
			fmt.Printf("✓ 匹配: %s -> %s\n", m.AnimeName, path)
		}
	}

	fmt.Printf("\n共匹配 %d 部动漫\n", matched)

	// 4. 保存新的 onedrive 映射表
	output, _ := json.MarshalIndent(onedriveMappings, "", "  ")
	os.WriteFile("../data/anime_mapping_onedrive.json", output, 0644)
	fmt.Println("已保存到 anime_mapping_onedrive.json")
}

func getOneDriveFolders() []string {
	cmd := exec.Command("rclone", "lsf", "onedrive:anime", "--dirs-only")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("rclone 失败:", err)
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var folders []string
	for _, line := range lines {
		line = strings.TrimSuffix(strings.TrimSpace(line), "/")
		if line != "" {
			folders = append(folders, line)
		}
	}
	return folders
}

func findInOneDrive(folderName string, onedriveFolders []string) (bool, string) {
	// 精确匹配
	for _, f := range onedriveFolders {
		if f == folderName {
			return true, f
		}
	}

	// 模糊匹配：去掉特殊字符后比较
	normalizedName := normalize(folderName)
	for _, f := range onedriveFolders {
		if normalize(f) == normalizedName {
			return true, f
		}
	}

	return false, ""
}

func normalize(s string) string {
	// 移除常见的特殊字符差异
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "´", "'")
	s = strings.ReplaceAll(s, "`", "'")
	s = strings.ReplaceAll(s, "³", "3")
	s = strings.ReplaceAll(s, "&", "&")
	return s
}
