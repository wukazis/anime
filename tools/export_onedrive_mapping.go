package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type AnimeMapping struct {
	AnimeName  string `json:"anime_name"`
	FolderName string `json:"folder_name"`
	FolderPath string `json:"folder_path"`
	FileID     string `json:"file_id"`
}

func main() {
	data, _ := os.ReadFile("../data/anime_mapping.json")
	var mappings []AnimeMapping
	json.Unmarshal(data, &mappings)

	var onedrive []AnimeMapping
	for _, m := range mappings {
		if strings.HasPrefix(m.FolderPath, "onedrive:") {
			onedrive = append(onedrive, m)
		}
	}

	output, _ := json.MarshalIndent(onedrive, "", "  ")
	os.WriteFile("../data/anime_mapping_onedrive.json", output, 0644)
	fmt.Printf("导出了 %d 条 OneDrive 映射\n", len(onedrive))
}
