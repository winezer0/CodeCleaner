package cleaner

import (
	"fmt"
	"github.com/winezer0/xutils/progress"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/winezer0/xutils/logging"
)

// JSFormater JS格式化清理器
type JSFormater struct {
	Path   string // 根目录路径
	DryRun bool   // 模拟模式
}

// NewJsCleaner 创建JS清理器实例
func NewJsCleaner(path string, dryRun bool) *JSFormater {
	return &JSFormater{
		Path:   path,
		DryRun: dryRun,
	}
}

// 使用 map 提高判断效率
var jsExtsMap = map[string]struct{}{
	".js":  {},
	".ts":  {},
	".jsx": {},
	".tsx": {},
	".mjs": {},
	".cjs": {},
}

// 判断函数示例
func isJSFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	_, exists := jsExtsMap[ext]
	return exists
}

// RunClean 执行JS格式化操作
func (c *JSFormater) RunClean() error {
	mode := getMode(c.DryRun)
	logging.Infof("start (%s) JS code formatting...", mode)

	// 验证根路径
	rootAbsPath, err := filepath.Abs(c.Path)
	if err != nil {
		return fmt.Errorf("invalid dir path: %v", err)
	}

	var count int
	var errorCount int

	// 初始化进度条
	bar := progress.NewSpinner(fmt.Sprintf("js format (%s) ...", mode))

	err = filepath.WalkDir(rootAbsPath, func(path string, d os.DirEntry, err error) error {
		// 更新进度条
		_ = bar.Add(1)
		bar.Describe(fmt.Sprintf("js format | handle: %d | error: %d", count, errorCount))

		if err != nil {
			_ = bar.Clear()
			logging.Warnf("access path error: %s - %v", path, err)
			errorCount++
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// 仅处理 js 文件
		if isJSFile(path) {
			count++
			if c.DryRun {
				// logging.Infof("[DryRun] Would format: %s", path) // Reduce log noise for progress bar
				return nil
			}

			// 执行 js-beautify
			cmd := exec.Command("js-beautify", "-r", path)
			output, err := cmd.CombinedOutput()
			if err != nil {
				logging.Errorf("js format error %s: %v, cmd output: %s", path, err, string(output))
				errorCount++
			} else {
				logging.Debugf("js format success: %s", path)
			}
			_ = bar.Clear()
		}
		return nil
	})

	_ = bar.Finish()

	logging.Infof("JS format completed: total count: %d , error count: %d", count, errorCount)
	return err
}
