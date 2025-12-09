package cmd

import (
	"context"
	"errors"
	"os"
	"runtime"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/cobra"
)

var (
	selfUpdateVersion      string
	selfUpdateDownloadURL  string
	selfUpdateDryRun       bool
	selfUpdateSkipChecksum bool
	selfUpdateForce        bool
)

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Download and replace with the latest release (checksum + backup by default)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		tag, err := utils.SelfUpdate(ctx, utils.SelfUpdateOptions{
			Owner:          "godspeedcurry",
			Repo:           "godscan",
			CurrentVersion: version,
			TargetVersion:  selfUpdateVersion,
			DownloadURL:    selfUpdateDownloadURL,
			DryRun:         selfUpdateDryRun,
			SkipChecksum:   selfUpdateSkipChecksum,
			Force:          selfUpdateForce,
			OS:             runtime.GOOS,
			Arch:           runtime.GOARCH,
			UserAgent:      "godscan/" + version,
			Token:          os.Getenv("GITHUB_TOKEN"),
		})
		if err != nil {
			if errors.Is(err, utils.ErrAlreadyLatest) {
				utils.Info("当前已是最新版本：%s", version)
				return nil
			}
			return err
		}
		if selfUpdateDryRun {
			utils.Info("最新版本：%s，运行不带 --dry-run 即可更新", tag)
			return nil
		}
		utils.Success("更新完成：%s", tag)
		return nil
	},
}

func init() {
	selfUpdateCmd.Flags().StringVar(&selfUpdateVersion, "version", "", "指定版本（缺省为最新）")
	selfUpdateCmd.Flags().StringVar(&selfUpdateDownloadURL, "download-url", "", "自定义下载 URL（跳过 GitHub 资产选择）")
	selfUpdateCmd.Flags().BoolVar(&selfUpdateDryRun, "dry-run", false, "仅检查可用版本，不写磁盘")
	selfUpdateCmd.Flags().BoolVar(&selfUpdateSkipChecksum, "skip-checksum", false, "跳过 SHA256 校验（不推荐）")
	selfUpdateCmd.Flags().BoolVar(&selfUpdateForce, "force", false, "忽略当前版本号，强制重新下载/覆盖")
	rootCmd.AddCommand(selfUpdateCmd)
}
