package main

import (
	"codecleaner/pkg/cleaner"
	"codecleaner/pkg/filestats"
	"os"

	"github.com/winezer0/slogs"

	"codecleaner/internal/utils"
)

func main() {
	// 打印命令行输入配置
	opts, _ := InitOptionsArgs(1)
	defer slogs.CloseAll()

	// 统计模式 目录大小统计
	if opts.StatsDir {
		if err := filestats.RunStatsDir(opts.Path); err != nil {
			fatalf("directory stats failed: %v", err)
		}
		os.Exit(0) // 显示后退出，不执行后续逻辑
	}

	// 统计模式 文件类型统计
	if opts.StatsExt {
		if err := filestats.RunStatsExt(opts.Path); err != nil {
			fatalf("suffix stats failed: %v", err)
		}
		os.Exit(0) // 显示后退出，不执行后续逻辑
	}

	// 按后缀进行清理
	if opts.PresetName != "" {
		preset := initPresetConfig(opts.PresetName, opts.ConfigPath)

		// 创建清理器并运行
		if preset != nil {
			if (opts.EnWhite && len(preset.Stored) > 0) || (!opts.EnWhite && len(preset.Stored)+len(preset.RmDirs) > 0) {
				suffixCleaner := cleaner.NewSuffixCleaner(opts.Path, *preset, opts.EnWhite, opts.DryRun)
				if err := suffixCleaner.RunClean(); err != nil {
					fatalf("failed to clean suffix file list: %v", err)
				}
			} else {
				fatalf("current preset (%s) has no valid data configured: %s", opts.PresetName, utils.ToJSON(preset))
			}
		} else {
			fatalf("failed to initialize detailed config for preset (%s)!", opts.PresetName)
		}
	}

	// JS 格式化
	if opts.JsBeautify {
		jsCleaner := cleaner.NewJSFormater(opts.Path, opts.DryRun, opts.JsWorkers)
		if err := jsCleaner.RunClean(); err != nil {
			fatalf("js formatting failed: %v", err)
		}
	}

	// 清理空白文件
	if opts.RmEmpty {
		emptyCleaner := cleaner.NewEmptyCleaner(opts.Path, opts.DryRun)
		if err := emptyCleaner.RunClean(); err != nil {
			fatalf("failed to clean empty file dirs: %v", err)
		}
	}
}
