package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// Bangumi 番剧数据
type Anime struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	NameCN   string   `json:"name_cn"`
	Year     int      `json:"year"`
	Cover    string   `json:"cover"`
	Summary  string   `json:"summary"`
	Rating   float64  `json:"rating"`
	Tags     []string `json:"tags"`
	Episodes int      `json:"episodes"`
}

// 映射记录
type AnimeMapping struct {
	Anime
	PikPakPath string `json:"pikpak_path"` // PikPak 上的路径
	FolderName string `json:"folder_name"` // 文件夹名
}

func main() {
	// 1. 从 matched_magnets.txt 读取番剧顺序
	animeOrder := loadAnimeOrder("../data/matched_magnets.txt")
	fmt.Printf("从磁链文件读取了 %d 部番剧顺序\n", len(animeOrder))

	// 2. 加载番剧数据库
	animeDB := loadAnimeDB("../data/anime_db.json")
	fmt.Printf("加载了 %d 部番剧数据\n", len(animeDB))

	// 3. 获取 PikPak 文件列表（按创建时间排序）
	pikpakFiles := listPikPakFiles("pikpak:wukazi")
	fmt.Printf("PikPak 上有 %d 个文件夹\n", len(pikpakFiles))

	// 4. 按顺序映射
	var mappings []AnimeMapping

	for i, animeName := range animeOrder {
		if i >= len(pikpakFiles) {
			fmt.Printf("⚠️ PikPak 文件数量不足，跳过: %s\n", animeName)
			continue
		}

		// 查找番剧数据
		anime := findAnime(animeDB, animeName)
		if anime == nil {
			fmt.Printf("⚠️ 未找到番剧数据: %s\n", animeName)
			continue
		}

		mapping := AnimeMapping{
			Anime:      *anime,
			PikPakPath: "wukazi/" + pikpakFiles[i].Name,
			FolderName: pikpakFiles[i].Name,
		}
		mappings = append(mappings, mapping)
		fmt.Printf("[%d] %s -> %s\n", i+1, animeName, pikpakFiles[i].Name)
	}

	// 5. 保存映射表
	saveMapping(mappings, "../data/anime_mapping.json")
	fmt.Printf("\n完成！映射了 %d 部番剧，保存到 data/anime_mapping.json\n", len(mappings))
}

// 从 matched_magnets.txt 读取番剧名顺序
func loadAnimeOrder(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("无法打开文件:", err)
		return nil
	}
	defer file.Close()

	var names []string
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			name := strings.TrimPrefix(line, "# ")
			names = append(names, name)
		}
	}
	return names
}

type PikPakFile struct {
	Name    string
	ModTime time.Time
}

func listPikPakFiles(remote string) []PikPakFile {
	cmd := exec.Command("rclone", "lsjson", remote)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("获取 PikPak 文件列表失败:", err)
		return nil
	}

	var files []struct {
		Name    string `json:"Name"`
		ModTime string `json:"ModTime"`
		IsDir   bool   `json:"IsDir"`
	}
	json.Unmarshal(output, &files)

	var result []PikPakFile
	for _, f := range files {
		if !f.IsDir {
			continue // 只要文件夹
		}
		t, _ := time.Parse(time.RFC3339, f.ModTime)
		result = append(result, PikPakFile{
			Name:    f.Name,
			ModTime: t,
		})
	}

	// 按创建时间排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].ModTime.Before(result[j].ModTime)
	})

	return result
}

func findAnime(animeDB []Anime, name string) *Anime {
	// 提取番剧名（去掉年份）
	// 格式: "犬夜叉 (2000)"
	parts := strings.Split(name, " (")
	searchName := parts[0]

	for i := range animeDB {
		if animeDB[i].NameCN == searchName || animeDB[i].Name == searchName {
			return &animeDB[i]
		}
	}
	return nil
}

func loadAnimeDB(filename string) []Anime {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("无法读取番剧数据库:", err)
		return nil
	}

	var animeList []Anime
	json.Unmarshal(data, &animeList)
	return animeList
}

func saveMapping(mappings []AnimeMapping, filename string) {
	data, _ := json.MarshalIndent(mappings, "", "  ")
	os.WriteFile(filename, data, 0644)
}
