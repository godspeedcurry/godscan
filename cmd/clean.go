package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type CleanOptions struct {
}

var (
	cleanOptions CleanOptions
)

// iconCmd represents the icon command

func init() {
	iconCmd := newCommandWithAliases("clean", "clean logs", []string{"cc"}, &cleanOptions)
	rootCmd.AddCommand(iconCmd)
}

func (o *CleanOptions) validateOptions() error {
	return nil
}

func (o *CleanOptions) run() {
	reader := bufio.NewReader(os.Stdin)

	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error fetching current directory:", err)
		return
	}

	// 正则表达式匹配形如YYYY-MM-DD的目录名
	dateDirRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

	err = filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Error accessing path:", path, err)
			return err
		}

		if info.IsDir() && dateDirRegex.MatchString(info.Name()) {
			// 检查是否删除目录
			fmt.Printf("Do you want to delete directory: %s? (y/N) ", path)
			response, _ := reader.ReadString('\n')
			if strings.TrimSpace(response) == "y" || strings.TrimSpace(response) == "" {
				if err := os.RemoveAll(path); err != nil {
					fmt.Println("Error deleting directory:", err)
				} else {
					fmt.Println("Directory deleted:", path)
				}
			}
			return filepath.SkipDir
		} else if !info.IsDir() {
			if strings.HasSuffix(info.Name(), ".log") || strings.HasSuffix(info.Name(), ".csv") {
				// 检查是否删除文件
				fmt.Printf("Do you want to delete file: %s? (y/N) ", path)
				response, _ := reader.ReadString('\n')
				if strings.TrimSpace(response) == "y" || strings.TrimSpace(response) == "" {
					if err := os.Remove(path); err != nil {
						fmt.Println("Error deleting file:", err)
					} else {
						fmt.Println("File deleted:", path)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking the path:", err)
	}
}
