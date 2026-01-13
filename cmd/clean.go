package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/godspeedcurry/godscan/utils"
)

type CleanOptions struct {
}

var (
	cleanOptions CleanOptions
)

// iconCmd represents the icon command

func init() {
	cleanCmd := newCommandWithAliases("clean", "Remove generated logs and dated folders", []string{"cc"}, &cleanOptions)
	rootCmd.AddCommand(cleanCmd)
}

func (o *CleanOptions) validateOptions() error {
	return nil
}

func (o *CleanOptions) run() {
	reader := bufio.NewReader(os.Stdin)

	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		utils.Error("Error fetching current directory: %v", err)
		return
	}

	// 正则表达式匹配形如YYYY-MM-DD的目录名
	dateDirRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

	err = filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			utils.Error("Error accessing path: %s %v", path, err)
			return err
		}

		// Skip source directories to prevent accidental deletion of assets
		if info.IsDir() {
			name := info.Name()
			if name == "utils" || name == "cmd" || name == "common" || name == ".git" || name == ".agent" || name == ".github" {
				return filepath.SkipDir
			}
		}

		if info.IsDir() && dateDirRegex.MatchString(info.Name()) {
			// 检查是否删除目录
			utils.Info("Do you want to delete directory: %s? (y/N)", path)
			response, _ := reader.ReadString('\n')
			if strings.TrimSpace(response) == "y" || strings.TrimSpace(response) == "" {
				if err := os.RemoveAll(path); err != nil {
					utils.Error("Error deleting directory: %v", err)
				} else {
					utils.Success("Directory deleted: %s", path)
				}
			}
			return filepath.SkipDir
		} else if !info.IsDir() {
			// Update: include html (reports), json (graph/summary), db (spider)
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if ext == ".log" || ext == ".csv" || ext == ".html" || ext == ".json" || ext == ".db" {
				// 检查是否删除文件
				utils.Info("Do you want to delete file: %s? (y/N)", path)
				response, _ := reader.ReadString('\n')
				// Fix: Default to No for safety
				if strings.TrimSpace(strings.ToLower(response)) == "y" {
					if err := os.Remove(path); err != nil {
						utils.Error("Error deleting file: %v", err)
					} else {
						utils.Success("File deleted: %s", path)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		utils.Error("Error walking the path: %v", err)
	}
}
