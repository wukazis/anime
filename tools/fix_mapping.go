package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// 从 PikPak 重新获取文件夹列表，修复映射表中的乱码

func init() {
	os.Setenv("HTTP_PROXY", "http://127.0.0.1:10101")
	os.Setenv("HTTPS_PROXY", "http://127.0.0.1:10101")
}

type AnimeMapping struct {
	AnimeName  string `json:"anime_name"`
	FolderName string `json:"folder_name"`
	FolderPath string `json:"folder_path"`
	FileID     string `json:"file_id"`
}

type PikPakFolder struct {
	Name string `json:"Name"`
	ID   string `json:"ID"`
}

func main() {
	// 加载当前映射表
	mappings := loadMappings("../data/anime_mapping.json")
	fmt.Printf("加载了 %d 条映射\n", len(mappings))

	// 从 PikPak 获取文件夹列表
	fmt.Println("从 PikPak 获取文件夹列表...")
	pikpakFolders := getPikPakFolders("pikpak:wukazi")
	fmt.Printf("PikPak 上有 %d 个文件夹\n", len(pikpakFolders))

	// 建立 ID -> 名称 的索引
	idToName := make(map[string]string)
	for _, f := range pikpakFolders {
		idToName[f.ID] = f.Name
	}

	// 建立简化名索引用于模糊匹配
	simplifiedToName := make(map[string]string)
	for _, f := range pikpakFolders {
		simplified := simplifyName(f.Name)
		simplifiedToName[simplified] = f.Name
	}

	// 修复映射表
	fixed := 0
	for i := range mappings {
		oldName := mappings[i].FolderName
		
		// 方法1: 通过 FileID 精确匹配
		if correctName, ok := idToName[mappings[i].FileID]; ok {
			if correctName != oldName {
				mappings[i].FolderName = correctName
				mappings[i].FolderPath = "wukazi/" + correctName
				fixed++
				fmt.Printf("✅ 修复: %s\n   旧: %s\n   新: %s\n", mappings[i].AnimeName, oldName, correctName)
				continue
			}
		}

		// 方法2: 通过简化名模糊匹配
		simplified := simplifyName(oldName)
		if correctName, ok := simplifiedToName[simplified]; ok {
			if correctName != oldName {
				mappings[i].FolderName = correctName
				mappings[i].FolderPath = "wukazi/" + correctName
				fixed++
				fmt.Printf("✅ 修复(模糊): %s\n   旧: %s\n   新: %s\n", mappings[i].AnimeName, oldName, correctName)
			}
		}
	}

	// 保存修复后的映射表
	saveMappings(mappings, "../data/anime_mapping.json")
	fmt.Printf("\n完成！修复了 %d 条映射\n", fixed)
}

func getPikPakFolders(remote string) []PikPakFolder {
	cmd := exec.Command("rclone", "lsjson", remote, "--dirs-only")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("获取 PikPak 文件夹列表失败:", err)
		return nil
	}

	var folders []PikPakFolder
	json.Unmarshal(output, &folders)
	return folders
}

func loadMappings(filename string) []AnimeMapping {
	data, _ := os.ReadFile(filename)
	var mappings []AnimeMapping
	json.Unmarshal(data, &mappings)
	return mappings
}

func saveMappings(mappings []AnimeMapping, filename string) {
	data, _ := json.MarshalIndent(mappings, "", "  ")
	os.WriteFile(filename, data, 0644)
}

func simplifyName(name string) string {
	var result strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		}
	}
	return result.String()
}
