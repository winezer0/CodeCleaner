package main

import (
	"codecleaner/internal/config"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/winezer0/slogs"

	"codecleaner/internal/utils"
)

// 版本信息常量（根据实际情况修改）
const (
	AppName      = "CodeCleaner"
	AppShortDesc = "code file cleaning tool"
	AppLongDesc  = "code file cleaning tool, cleans non-code files in specified directory"
	AppVersion   = "0.1.3"
	BuildDate    = "2026-08-03"
)

// Options command line options
type Options struct {
	Path       string `short:"p" long:"path" description:"scan start directory path"`
	PresetName string `short:"P" long:"preset" description:"use preset rules or ext/dir: comma-separated suffix list (e.g., ext: exe,txt)" default:"common"`
	ConfigPath string `short:"c" long:"config" description:"custom yaml config file path"`

	// GenerateConfig 生成默认配置文件
	ShowPresetList bool `long:"list" description:"list all presets"`
	GenerateConfig bool `long:"gen" description:"gen default config to <ConfigPath>"`

	DryRun     bool `short:"d" long:"dry_run" description:"preview mode: show files to be deleted, do not execute deletion"`
	EnWhite    bool `short:"w" long:"en_white" description:"whitelist mode: only keep files with suffixes specified in stored preset"`
	RmEmpty    bool `short:"e" long:"rm_empty" description:"remove empty files: remove empty directories and file paths when enabled"`
	JsBeautify bool `short:"j" long:"js-beautify" description:"format js: format js files with pure go jsbeautify library"`
	JsWorkers  int  `short:"J" long:"js-workers" description:"number of concurrent goroutines for js formatting" default:"4"`

	// 统计信息显示
	StatsExt bool `short:"s" long:"stats_ext" description:"enable stats mode: show distribution of file quantities by suffix"`
	StatsDir bool `short:"S" long:"stats_dir" description:"enable stats mode: show distribution of file quantities by directory"`
	Version  bool `short:"v" long:"version" description:"output version information"`

	// Log configuration
	LogFile       string `long:"lf" description:"Log file path (default: null)"`
	LogLevel      string `long:"ll" description:"Log level (debug/info/warn/error)" default:"info"`
	ConsoleFormat string `long:"cf" description:"Console log format (T L C M F combination or off|null to disable)" default:"M"`
}

// fatalf 记录错误日志并退出程序（slogs 不提供 Fatalf，封装等价行为）
func fatalf(format string, args ...any) {
	slogs.Errorf(format, args...)
	os.Exit(1)
}

// InitOptionsArgs 常用的工具函数，解析parser和logging配置
func InitOptionsArgs(minimumParams int) (*Options, *flags.Parser) {
	opts := &Options{}
	parser := flags.NewParser(opts, flags.Default)
	parser.Name = AppName
	parser.Usage = "[OPTIONS]"
	parser.ShortDescription = AppShortDesc
	parser.LongDescription = AppLongDesc

	// 命令行参数数量检查 指不包含程序名本身的参数数量
	if minimumParams > 0 && len(os.Args)-1 < minimumParams {
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	// 命令行参数解析检查
	if _, err := parser.Parse(); err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && errors.Is(flagsErr.Type, flags.ErrHelp) {
			os.Exit(0)
		}
		fmt.Printf("Error:%v\n", err)
		os.Exit(1)
	}

	// 版本号输出
	if opts.Version {
		fmt.Printf("%s version %s\n", AppName, AppVersion)
		fmt.Printf("Build Date: %s\n", BuildDate)
		os.Exit(0)
	}

	// 初始化日志器
	logCfg := slogs.NewConfig(opts.LogLevel, opts.LogFile, opts.ConsoleFormat)
	if err := slogs.Init(logCfg); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// 处理生成配置文件命令
	if opts.GenerateConfig {
		configPath := opts.ConfigPath
		if configPath == "" {
			configPath = AppName + ".yaml"
		}
		if err := config.GenDefaultConfig(configPath); err != nil {
			fatalf("Failed to generate config file: %v", err)
		}
		slogs.Infof("Default config file has been generated: %s", configPath)
		os.Exit(0)
	}

	// 输出当前配置文件中的所有preset名称
	if opts.ShowPresetList {
		conf, err := config.LoadConfig(opts.ConfigPath, AppName)
		if err != nil {
			fatalf("load config: %s error: %v", conf, err)
		}
		config.PrintPresetSummary(conf)
		os.Exit(0)
	}

	// 检查是否输入 Path
	if opts.Path == "" {
		fatalf("code file directory must be specified!!!")
	}

	return opts, parser
}

func initPresetConfig(presetStr string, presetFile string) *config.PresetConfig {
	// 获取preset配置
	var preset *config.PresetConfig
	if strings.Contains(presetStr, "ext:") || strings.Contains(presetStr, "dir:") {
		// 从输入命令行中解析出 preset
		extList, dirList := parseCmdExtDir(presetStr)
		extList = utils.SliceUnique(utils.ToLowerKeys(extList), true, true)
		dirList = utils.SliceUnique(utils.ToLowerKeys(dirList), true, true) // 仅在黑名单模式下有效,用于删除自定义目录，很少用
		preset = config.NewPresetConfig("temp list", extList, extList, dirList)
		slogs.Infof("cmd init preset: %s", utils.ToJSON(preset))
	} else {
		conf, err := config.LoadConfig(presetFile, AppName)
		if err != nil {
			slogs.Errorf("load config: %s error: %v", conf, err)
		} else {
			if preset, _ = conf.GetPreset(presetStr); preset == nil {
				slogs.Errorf("config %s not contain key: %s and custom preset not like (like ext:xxx,xxx)", conf, presetStr)
			}
		}
	}
	return preset
}

// parseCmdExtDir 解析和格式化命令行參數中的dir和ext参数
func parseCmdExtDir(input string) (extList, dirList []string) {
	extList = []string{}
	dirList = []string{}

	// 兼容低版本 Go 的正则：不使用零宽断言，而是匹配到下一个标记或结尾
	// 模式说明：
	// (ext|dir):   匹配 ext: 或 dir:
	// (.*?)        非贪婪匹配内容（直到下一个标记或结尾）
	// (?:ext:|dir:|$)  匹配下一个标记（非捕获组）或字符串结尾
	re := regexp.MustCompile(`(ext|dir):(.*?)(ext:|dir:|$)`)

	remaining := input // 剩余未处理的字符串
	for {
		// 查找匹配
		match := re.FindStringSubmatch(remaining)
		if len(match) != 4 {
			break // 无更多匹配，退出循环
		}

		key := strings.ToLower(match[1])
		value := strings.TrimSpace(match[2])
		nextMarker := match[3] // 下一个标记（可能为空，即到结尾）

		// 处理当前值
		items := strings.Split(value, ",")
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item != "" {
				switch key {
				case "ext":
					extList = append(extList, item)
				case "dir":
					dirList = append(dirList, item)
				}
			}
		}

		// 移动到下一个标记的位置继续处理
		remaining = remaining[len(match[1])+1+len(match[2]):] // +1 是因为 key 后面有个冒号
		if nextMarker == "" {
			break // 已到结尾，退出
		}
	}

	return extList, dirList
}
