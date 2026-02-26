package main

import (
	"codecleaner/internal/config"
	"codecleaner/internal/embeds"
	"codecleaner/pkg/cleaner"
	"codecleaner/pkg/filestats"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/winezer0/xutils/logging"
	"github.com/winezer0/xutils/utils"
)

// 版本信息常量（根据实际情况修改）
const (
	AppName      = "CodeCleaner"
	AppShortDesc = "code file cleaning tool"
	AppLongDesc  = "code file cleaning tool, cleans non-code files in specified directory"
	AppVersion   = "0.1.0"
	BuildDate    = "2026-02-27"
)

// Options command line options
type Options struct {
	Path         string `short:"p" long:"path" description:"scan start directory path"`
	Preset       string `short:"P" long:"preset" description:"use preset cleaning rules (default: common) or ext/dir: comma-separated suffix list (e.g., ext: exe,txt)"`
	PresetConfig string `short:"c" long:"preset_config" description:"custom yaml config file path" default:"cleaner.yaml"`

	DryRun     bool `short:"d" long:"dry_run" description:"preview mode: show files to be deleted, do not execute deletion"`
	EnWhite    bool `short:"w" long:"en_white" description:"whitelist mode: only keep files with suffixes specified in stored preset"`
	RmEmpty    bool `short:"e" long:"rm_empty" description:"remove empty files: remove empty directories and file paths when enabled"`
	JsBeautify bool `short:"j" long:"js-beautify" description:"format js: call js-beautify to format js files"`
	JsWorkers  int  `short:"J" long:"js-workers" description:"number of concurrent workers for js formatting" default:"4"`

	// 统计信息显示
	StatsExt bool `short:"s" long:"stats_ext" description:"enable stats mode: show distribution of file quantities by suffix"`
	StatsDir bool `short:"S" long:"stats_dir" description:"enable stats mode: show distribution of file quantities by directory"`
	Version  bool `short:"v" long:"version" description:"output version information"`

	// Log configuration
	LogFile       string `long:"lf" description:"Log file path (default: null)"`
	LogLevel      string `long:"ll" description:"Log level (debug/info/warn/error)" default:"info"`
	ConsoleFormat string `long:"cf" description:"Console log format (T L C M F combination or off|null to disable)" default:"M"`
}

func main() {
	// 打印命令行输入配置
	opts, _ := InitOptionsArgs(1)

	// 统计模式 目录大小统计
	if opts.StatsDir {
		if err := filestats.RunStatsDir(opts.Path); err != nil {
			logging.Fatalf("directory stats failed: %v", err)
		}
		os.Exit(0) // 显示后退出，不执行后续逻辑
	}

	// 统计模式 文件类型统计
	if opts.StatsExt {
		if err := filestats.RunStatsExt(opts.Path); err != nil {
			logging.Fatalf("suffix stats failed: %v", err)
		}
		os.Exit(0) // 显示后退出，不执行后续逻辑
	}

	// 按后缀进行清理
	if opts.Preset != "" {
		preset := initPresetConfig(opts.Preset, opts.PresetConfig)

		// 创建清理器并运行
		if preset != nil {
			if (opts.EnWhite && len(preset.Stored) > 0) || (!opts.EnWhite && len(preset.Stored)+len(preset.RmDirs) > 0) {
				suffixCleaner := cleaner.NewSuffixCleaner(opts.Path, *preset, opts.EnWhite, opts.DryRun)
				if err := suffixCleaner.RunClean(); err != nil {
					logging.Fatalf("failed to clean suffix file list: %v", err)
				}
			} else {
				logging.Fatalf("current preset (%s) has no valid data configured: %s", opts.Preset, utils.ToJSON(preset))
			}
		} else {
			logging.Fatalf("failed to initialize detailed config for preset (%s)!", opts.Preset)
		}
	}

	// JS 格式化
	if opts.JsBeautify {
		jsCleaner := cleaner.NewJSFormater(opts.Path, opts.DryRun, opts.JsWorkers)
		if err := jsCleaner.RunClean(); err != nil {
			logging.Fatalf("js formatting failed: %v", err)
		}
	}

	// 清理空白文件
	if opts.RmEmpty {
		emptyCleaner := cleaner.NewEmptyCleaner(opts.Path, opts.DryRun)
		if err := emptyCleaner.RunClean(); err != nil {
			logging.Fatalf("failed to clean empty file dirs: %v", err)
		}
	}
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
	logCfg := logging.NewLogConfig(opts.LogLevel, opts.LogFile, opts.ConsoleFormat)
	if err := logging.InitLogger(logCfg); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync()

	// 检查 js-beautify 依赖
	if opts.JsBeautify {
		if err := exec.Command("js-beautify", "--version").Run(); err != nil {
			logging.Fatalf("js-beautify command not found, please install: npm install -g js-beautify")
		}
	}

	// 检查是否输入 Path
	if opts.Path == "" {
		logging.Fatalf("code file directory must be specified!!!")
	}

	return opts, parser
}

func initPresetConfig(presetStr string, presetFile string) *config.PresetConfig {
	// 获取preset配置
	var preset *config.PresetConfig
	if strings.Contains(presetStr, "ext:") || strings.Contains(presetStr, "dir:") {
		// 从输入命令行中解析出 preset
		extList, dirList := parseCmdExtDir(presetStr)
		extList = utils.UniqueSlice(utils.ToLowerKeys(extList), true, true)
		dirList = utils.UniqueSlice(utils.ToLowerKeys(dirList), true, true) // 仅在黑名单模式下有效,用于删除自定义目录，很少用
		preset = config.NewPresetConfig("temp list", extList, extList, dirList)
		logging.Infof("cmd init preset: %s", utils.ToJSON(preset))
	} else {
		// 从配置文件中获取 preset
		if utils.IsEmptyFile(presetFile) {
			utils.WriteToFile(presetFile, embeds.GetConfig())
			logging.Debugf("Success creat config from embed: %v", presetFile)
		}
		if conf, err := config.LoadConfig(presetFile); err != nil {
			logging.Errorf("load config: %s error: %v", conf, err)
		} else if preset, _ = conf.GetPreset(presetStr); preset == nil {
			logging.Errorf("config %s not contain key: %s and custom preset not like (like ext:xxx,xxx)", conf, presetStr)
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
