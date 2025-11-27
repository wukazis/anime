package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type AnimeInfo struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	NameCN  string   `json:"name_cn"`
	Year    int      `json:"year"`
	Date    string   `json:"date"`
	Summary string   `json:"summary"`
	Cover   string   `json:"cover"`
	Score   float64  `json:"score"`
	Tags    []string `json:"tags"`
}

func main() {
	data, _ := os.ReadFile("../data/anime_db.json")
	var animes []AnimeInfo
	json.Unmarshal(data, &animes)

	var output string
	for _, a := range animes {
		name := a.NameCN
		if name == "" {
			name = a.Name
		}
		output += fmt.Sprintf("%s %s\n", name, a.Date)
	}

	os.WriteFile("../data/anime_list.txt", []byte(output), 0644)
	fmt.Printf("已导出 %d 条到 data/anime_list.txt\n", len(animes))
}
