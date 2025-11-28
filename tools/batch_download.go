package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Task struct {
	Name   string
	Magnet string
}

type FailedTask struct {
	Name   string
	Magnet string
	Reason string
}

const (
	proxy       = "http://127.0.0.1:10101"
	downloadDir = "D:/Downloads/anime"
	timeout     = 5 * time.Minute  // 单个任务超时
	maxConcurrent = 3              // 并发下载数
)

func main() {
	tasks := loadTasks("../data/matched_magnets.txt")
	fmt.Printf("共加载 %d 个下载任务\n", len(tasks))

	var failed []FailedTask
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrent)

	for i, task := range tasks {
		wg.Add(1)
		sem <- struct{}{}
		
		go func(idx int, t Task) {
			defer wg.Done()
			defer func() { <-sem }()
			
			fmt.Printf("[%d/%d] 开始下载: %s\n", idx+1, len(tasks), t.Name)
			err := downloadWithAria2(t)
			if err != nil {
				mu.Lock()
				failed = append(failed, FailedTask{
					Name:   t.Name,
					Magnet: t.Magnet,
					Reason: err.Error(),
				})
				mu.Unlock()
				fmt.Printf("[%d/%d] ❌ 失败: %s - %v\n", idx+1, len(tasks), t.Name, err)
			} else {
				fmt.Printf("[%d/%d] ✅ 完成: %s\n", idx+1, len(tasks), t.Name)
			}
		}(i, task)
	}

	wg.Wait()

	// 保存失败列表
	saveFailedTasks(failed)
	fmt.Printf("\n下载完成！成功: %d, 失败: %d\n", len(tasks)-len(failed), len(failed))
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
	scanner.Buffer(buf, 10*1024*1024) // 支持超长行

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

func downloadWithAria2(task Task) error {
	// 创建以番剧名命名的目录
	dir := downloadDir + "/" + sanitizeName(task.Name)
	os.MkdirAll(dir, 0755)

	args := []string{
		"--all-proxy=" + proxy,
		"--dir=" + dir,
		"--seed-time=0",           // 下载完不做种
		"--bt-stop-timeout=180",   // 3分钟无速度则停止
		"--bt-tracker-connect-timeout=10",
		"--bt-tracker-timeout=10",
		"--max-tries=3",
		"--retry-wait=5",
		"--timeout=60",
		"--connect-timeout=30",
		"--bt-metadata-only=false",
		"--bt-save-metadata=false",
		"--follow-torrent=mem",
		task.Magnet,
	}

	cmd := exec.Command("aria2c", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		cmd.Process.Kill()
		return fmt.Errorf("下载超时")
	}
}

func sanitizeName(name string) string {
	// 移除不合法的文件名字符
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	return replacer.Replace(name)
}

func saveFailedTasks(failed []FailedTask) {
	if len(failed) == 0 {
		fmt.Println("没有失败的任务")
		return
	}

	file, err := os.Create("../data/download_failed.txt")
	if err != nil {
		fmt.Println("无法创建失败列表文件:", err)
		return
	}
	defer file.Close()

	for _, f := range failed {
		file.WriteString(fmt.Sprintf("# %s\n# 原因: %s\n%s\n\n", f.Name, f.Reason, f.Magnet))
	}
	fmt.Printf("失败列表已保存到 data/download_failed.txt\n")
}
