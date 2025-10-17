package main

import (
	"codecleaner/internal/config"
	"codecleaner/internal/embeds"
	"codecleaner/pkg/cleaner"
	"codecleaner/pkg/fileutils"
	"codecleaner/pkg/logging"
	"codecleaner/pkg/stats"
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
)

// Options command line options
type Options struct {
	Path   string `short:"p" long:"path" description:"扫描起始目录路径" required:"true"`
	Preset string `short:"r" long:"preset" description:"使用预设清理规则（默认common）" default:"common"`
	Config string `short:"c" long:"config" description:"自定义 YAML 配置文件路径" default:".cleaner.yaml"`
	Stats  bool   `short:"s" long:"stats" description:"启用统计模式：显示目录下所有文件的类型数量分布"`
	Try    bool   `short:"t" long:"try" description:"预览尝试模式：显示将删除的文件，不执行删除"`
	White  bool   `short:"w" long:"white" description:"白名单模式：仅保留预设中 stored 指定的文件后缀类型"`

	// Log configuration
	LogFile       string `long:"lf" description:"Log file path (default: null)"`
	LogLevel      string `long:"ll" description:"Log level (debug/info/warn/error)" default:"info"`
	ConsoleFormat string `long:"cf" description:"Console log format (T L C M F combination or off|null to disable)" default:"M"`
}

func main() {
	var opts Options

	parser := flags.NewParser(&opts, flags.Default)
	parser.Usage = "[OPTIONS]"

	// Custom help information
	parser.LongDescription = `代码文件清理工具 - 用于清理指定目录中的非代码文件`

	if _, err := parser.Parse(); err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && errors.Is(flagsErr.Type, flags.ErrHelp) {
			return
		}
		fmt.Sprintf("命令行参数解析错误: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logCfg := logging.NewLogConfig(opts.LogLevel, opts.LogFile, opts.ConsoleFormat)
	if err := logging.InitLogger(logCfg); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync()

	// 进行参数信息检查
	if opts.Try && opts.Preset == "" {
		logging.Fatalf("错误: --try 模式需要指定 --deleted 或 --preset 参数\n")
	}

	if opts.White && opts.Preset == "" {
		logging.Fatalf("错误: --white 模式需要指定 --preset 参数\n")
	}

	// 如果是统计模式，先调用目录统计，再调用文件类型统计
	if opts.Stats {
		if err := stats.RunDirsStats(opts.Path); err != nil {
			logging.Fatalf("目录统计操作失败: %v", err)
		}
		if err := stats.RunStats(opts.Path); err != nil {
			logging.Fatalf("统计操作失败: %v", err)
		}
		return
	}

	//生成默认配置文件
	if fileutils.IsEmptyFile(opts.Config) {
		fileutils.MakeDirs(opts.Config, true)
		fileutils.WriteAny(opts.Config, embeds.GetConfig())
		logging.Debugf("Success creat config from embed: %v", opts.Config)
	}

	// 准备清理参数
	if opts.Preset != "" && opts.Config != "" {
		// 从config中获取
		cfg, err := config.LoadConfig(opts.Config)
		if err != nil {
			logging.Fatalf("load config: %s error: %v", opts.Config, err)
		}

		preset, exists := cfg.GetPreset(opts.Preset)
		if !exists {
			logging.Fatalf("config file %s not contain key: %s error: %v", opts.Config, opts.Preset, err)
		}

		// 创建清理器并运行
		cleaner := cleaner.NewCleaner(opts.Path, *preset, opts.Try, opts.White)
		if err := cleaner.RunClean(); err != nil {
			logging.Fatalf("清理操作失败: %v", err)
		}
	}

}
