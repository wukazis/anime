package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	file, err := os.Open("../data/matched_magnets.txt")
	if err != nil {
		fmt.Println("无法打开文件:", err)
		return
	}
	defer file.Close()

	out, _ := os.Create("magnets_for_js.txt")
	defer out.Close()

	out.WriteString("const magnets = [\n")

	var currentName string
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)
	count := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			currentName = strings.TrimPrefix(line, "# ")
			// 转义引号
			currentName = strings.ReplaceAll(currentName, `"`, `\"`)
		} else if strings.HasPrefix(line, "magnet:") && currentName != "" {
			// 只取磁力链接的前200个字符（包含hash即可）
			magnet := line
			if len(magnet) > 200 {
				// 找到第一个 &tr= 截断
				if idx := strings.Index(magnet, "&tr="); idx > 0 {
					magnet = magnet[:idx]
				}
			}
			out.WriteString(fmt.Sprintf(`  { name: "%s", url: "%s" },`+"\n", currentName, magnet))
			count++
		}
	}

	out.WriteString("];\n")
	fmt.Printf("已导出 %d 个磁力链接到 magnets_for_js.txt\n", count)
}
