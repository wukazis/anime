package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// 只做精确匹配，更新已传输到 OneDrive 的文件夹路径

type AnimeMapping struct {
	AnimeName  string `json:"anime_name"`
	FolderName string `json:"folder_name"`
	FolderPath string `json:"folder_path"`
	FileID     string `json:"file_id"`
}

type OneDriveFolder struct {
	Name string `json:"Name"`
}

func main() {
	// 加载当前映射表
	mappings := loadMappings("../data/anime_mapping.json")
	fmt.Printf("加载了 %d 条映射\n", len(mappings))

	// 加载 OneDrive 文件夹列表
	odFolders := loadOneDriveFolders("../data/onedrive_folders_utf8.json")
	fmt.Printf("OneDrive 上有 %d 个文件夹\n", len(odFolders))

	// 建立 OneDrive 文件夹名索引（只做精确匹配）
	odIndex := make(map[string]bool)
	for _, f := range odFolders {
		odIndex[f.Name] = true
	}

	// 更新映射表（只更新 folder_path，不改 folder_name）
	updated := 0
	for i := range mappings {
		folderName := mappings[i].FolderName

		// 精确匹配
		if odIndex[folderName] {
			if !strings.HasPrefix(mappings[i].FolderPath, "onedrive:") {
				mappings[i].FolderPath = "onedrive:anime/" + folderName
				updated++
				fmt.Printf("✅ %s\n", mappings[i].AnimeName)
			}
		}
	}

	// 找出未匹配的 OneDrive 文件夹
	mappingIndex := make(map[string]bool)
	for _, m := range mappings {
		mappingIndex[m.FolderName] = true
	}
	fmt.Println("\n未匹配的 OneDrive 文件夹:")
	for _, f := range odFolders {
		if !mappingIndex[f.Name] {
			fmt.Printf("  - %s\n", f.Name)
		}
	}

	// 保存更新后的映射表
	saveMappings(mappings, "../data/anime_mapping.json")
	fmt.Printf("\n完成！更新了 %d 条映射\n", updated)
}

func loadMappings(filename string) []AnimeMapping {
	data, _ := os.ReadFile(filename)
	var mappings []AnimeMapping
	json.Unmarshal(data, &mappings)
	return mappings
}

func loadOneDriveFolders(filename string) []OneDriveFolder {
	data, _ := os.ReadFile(filename)
	// 去除 UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xef && data[1] == 0xbb && data[2] == 0xbf {
		data = data[3:]
	}
	var folders []OneDriveFolder
	json.Unmarshal(data, &folders)
	return folders
}

func saveMappings(mappings []AnimeMapping, filename string) {
	data, _ := json.MarshalIndent(mappings, "", "  ")
	os.WriteFile(filename, data, 0644)
}
