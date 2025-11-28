package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	file, _ := os.Open("../data/matched_magnets.txt")
	defer file.Close()

	var magnets []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "magnet:") {
			magnets = append(magnets, line)
		}
	}

	// 只保存磁力链接，每行一个
	output := strings.Join(magnets, "\n")
	os.WriteFile("../data/magnets_only.txt", []byte(output), 0644)

	fmt.Printf("已提取 %d 条磁力链接到 data/magnets_only.txt\n", len(magnets))
}
