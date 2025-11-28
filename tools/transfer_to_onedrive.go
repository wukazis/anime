package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// 设置代理
func init() {
	os.Setenv("HTTP_PROXY", "http://127.0.0.1:10101")
	os.Setenv("HTTPS_PROXY", "http://127.0.0.1:10101")
}

// 从 PikPak 移动文件到 OneDrive

const (
	srcRemote = "pikpak:wukazi"
	dstRemote = "onedrive:anime" // OneDrive 目标文件夹
)

type FolderInfo struct {
	Name  string `json:"Name"`
	IsDir bool   `json:"IsDir"`
}

var logFile *os.File

func main() {
	var err error
	logFile, err = os.Create("../data/transfer_log.txt")
	if err != nil {
		fmt.Println("无法创建日志文件:", err)
		return
	}
	defer logFile.Close()

	log("开始从 PikPak 移动到 OneDrive")
	log(fmt.Sprintf("源: %s", srcRemote))
	log(fmt.Sprintf("目标: %s", dstRemote))

	// 获取 PikPak 文件夹列表
	folders := listFolders(srcRemote)
	log(fmt.Sprintf("共 %d 个文件夹需要移动", len(folders)))

	var success, failed int
	for i, folder := range folders {
		log(fmt.Sprintf("[%d/%d] 移动: %s", i+1, len(folders), folder))

		src := srcRemote + "/" + folder
		dst := dstRemote + "/" + folder

		err := moveFolder(src, dst)
		if err != nil {
			log(fmt.Sprintf("  ❌ 失败: %v", err))
			failed++
		} else {
			log(fmt.Sprintf("  ✅ 成功"))
			success++
		}
	}

	log(fmt.Sprintf("\n完成！成功: %d, 失败: %d", success, failed))
}

func log(msg string) {
	fmt.Println(msg)
	logFile.WriteString(time.Now().Format("2006-01-02 15:04:05") + " " + msg + "\n")
}

func listFolders(remote string) []string {
	cmd := exec.Command("rclone", "lsjson", remote)
	output, err := cmd.Output()
	if err != nil {
		log("获取文件夹列表失败: " + err.Error())
		return nil
	}

	var items []FolderInfo
	json.Unmarshal(output, &items)

	var folders []string
	for _, item := range items {
		if item.IsDir {
			folders = append(folders, item.Name)
		}
	}
	return folders
}

func moveFolder(src, dst string) error {
	// 使用 rclone move，高并发跑满带宽
	cmd := exec.Command("rclone", "move", src, dst,
		"--progress",
		"--transfers", "16",        // 同时传输16个文件
		"--checkers", "32",         // 32个检查线程
		"--multi-thread-streams", "8", // 每个文件8线程
		"--buffer-size", "64M",     // 64M缓冲
		"--retries", "3",
		"--low-level-retries", "10",
		"-v",
	)

	// 实时输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	// 移动成功后删除空文件夹
	exec.Command("rclone", "rmdir", src).Run()
	return nil
}
