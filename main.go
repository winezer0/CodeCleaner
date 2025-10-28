package main

import (
	"codecleaner/internal/config"
	"codecleaner/internal/embeds"
	"codecleaner/pkg/cleaner"
	"codecleaner/pkg/cmdutils"
	"codecleaner/pkg/filestats"
	"codecleaner/pkg/fileutils"
	"codecleaner/pkg/logging"
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
	"strings"
)

// 版本信息常量（根据实际情况修改）
const (
	AppName      = "CodeCleaner"
	AppShortDesc = "ISEC 代码文件清理工具"
	AppLongDesc  = "ISEC 代码文件清理工具, 清理指定目录中的非代码文件"
	AppVersion   = "0.0.7"
	BuildDate    = "2025-10-28"
)

// Options command line options
type Options struct {
	Path         string `short:"p" long:"path" description:"扫描起始目录路径"`
	Preset       string `short:"P" long:"preset" description:"使用预设清理规则(默认common) 或 ext/dir:逗号分割的后缀列表 (如ext: exe,txt)"`
	PresetConfig string `short:"c" long:"preset_config" description:"自定义 YAML 配置文件路径" default:"cleaner.yaml"`

	DryRun  bool `short:"d" long:"dry_run" description:"预览尝试模式：显示将删除的文件，不执行删除"`
	EnWhite bool `short:"w" long:"en_white" description:"白名单模式：仅保留预设中 stored 指定的文件后缀类型"`
	RmEmpty bool `short:"e" long:"rm_empty" description:"移除空文件：启用时移除空目录和空文件路径"`

	// 统计信息显示
	StatsExt bool `short:"s" long:"stats_ext" description:"启用统计模式：显示目录下(后缀类型) 数量分布"`
	StatsDir bool `short:"S" long:"stats_dir" description:"启用统计模式：显示目录下(目录文件) 数量分布"`
	Version  bool `short:"v" long:"version" description:"输出版本信息"`

	// Log configuration
	LogFile       string `long:"lf" description:"Log file path (default: null)"`
	LogLevel      string `long:"ll" description:"Log level (debug/info/warn/error)" default:"info"`
	ConsoleFormat string `long:"cf" description:"Console log format (T L C M F combination or off|null to disable)" default:"M"`
}

func main() {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	// 添加描述信息
	parser.Name = AppName
	parser.Usage = "[OPTIONS]"
	parser.ShortDescription = AppShortDesc
	parser.LongDescription = AppLongDesc

	if _, err := parser.Parse(); err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && errors.Is(flagsErr.Type, flags.ErrHelp) {
			return
		}
		fmt.Printf("命令行参数解析错误: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logCfg := logging.NewLogConfig(opts.LogLevel, opts.LogFile, opts.ConsoleFormat)
	if err := logging.InitLogger(logCfg); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync()

	// 新增：判断是否需要显示版本信息
	if opts.Version {
		fmt.Printf("CodeClear version %s\n", AppVersion)
		fmt.Printf("Build Date: %s\n", BuildDate)
		os.Exit(0) // 显示后退出，不执行后续逻辑
	}

	// 检查是否输入 Path
	if opts.Path == "" {
		logging.Fatalf("必须有指定代码文件所在目录!!!")
	}

	// 统计模式 目录大小统计
	if opts.StatsDir {
		if err := filestats.RunStatsDir(opts.Path); err != nil {
			logging.Fatalf("目录统计操作失败: %v", err)
		}
		os.Exit(0) // 显示后退出，不执行后续逻辑
	}

	// 统计模式 文件类型统计
	if opts.StatsExt {
		if err := filestats.RunStatsExt(opts.Path); err != nil {
			logging.Fatalf("后缀统计操作失败: %v", err)
		}
		os.Exit(0) // 显示后退出，不执行后续逻辑
	}

	// 按后缀进行清理
	if opts.Preset != "" {
		preset := initPresetConfig(opts.Preset, opts.PresetConfig)

		// 创建清理器并运行
		if preset != nil {
			if (opts.EnWhite && len(preset.Stored) > 0) || (!opts.EnWhite && len(preset.Stored)+len(preset.RmDirs) > 0) {
				suffixCleaner := cleaner.NewCleaner(opts.Path, *preset, opts.EnWhite, opts.DryRun)
				if err := suffixCleaner.RunClean(); err != nil {
					logging.Fatalf("清理后缀文件列表失败: %v", err)
				}
			} else {
				logging.Fatalf("当前 Preset (%s) 未配置有效数据: %s", opts.Preset, cmdutils.AnyToJson(preset))
			}
		} else {
			logging.Fatalf("当前 Preset (%s) 配置初始化详细配置失败!", opts.Preset)
		}
	}

	// 清理空白文件
	if opts.RmEmpty {
		emptyCleaner := cleaner.NewEmptyCleaner(opts.Path, opts.DryRun)
		if err := emptyCleaner.RunClean(); err != nil {
			logging.Fatalf("清理空白文件目录失败: %v", err)
		}
	}
}

func initPresetConfig(presetStr string, presetFile string) *config.PresetConfig {
	// 获取preset配置
	var preset *config.PresetConfig
	if strings.Contains(presetStr, "ext:") || strings.Contains(presetStr, "dir:") {
		// 从输入命令行中解析出 preset
		extList, dirList := cmdutils.ParseCmdExtDir(presetStr)
		extList = cmdutils.ListUnique(extList, true)
		dirList = cmdutils.ListUnique(dirList, true) // 仅在黑名单模式下有效,用于删除自定义目录，很少用
		preset = config.NewPresetConfig("临时名单", extList, extList, dirList)
		logging.Infof("cmd init preset: %s", cmdutils.AnyToJson(preset))
	} else {
		// 从配置文件中获取 preset
		checkAndInitPresetFile(presetFile)
		if conf, err := config.LoadConfig(presetFile); err != nil {
			logging.Errorf("load config: %s error: %v", conf, err)
		} else if preset, _ = conf.GetPreset(presetStr); preset == nil {
			logging.Errorf("config %s not contain key: %s and custom preset not like (like ext:xxx,xxx)", conf, presetStr)
		}
	}
	return preset
}

// checkAndInitPresetFile presetFile为空时生成默认配置
func checkAndInitPresetFile(presetFile string) {
	if fileutils.IsEmptyFile(presetFile) {
		fileutils.MakeDirs(presetFile, true)
		fileutils.WriteAny(presetFile, embeds.GetConfig())
		logging.Debugf("Success creat config from embed: %v", presetFile)
	}
}
