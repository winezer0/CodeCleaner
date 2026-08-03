package cleaner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/winezer0/jsbeautify"
	"github.com/winezer0/xutils/progress"

	"github.com/winezer0/xutils/logging"
)

// JSFormater JS格式化清理器
type JSFormater struct {
	Path    string // 根目录路径
	DryRun  bool   // 模拟模式
	Workers int    // 并发工作线程数
}

// NewJSFormater 创建JS清理器实例
func NewJSFormater(path string, dryRun bool, workers int) *JSFormater {
	if workers <= 0 {
		workers = 1
	}
	return &JSFormater{
		Path:    path,
		DryRun:  dryRun,
		Workers: workers,
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

// formatJSFile 使用纯 Go 库格式化单个 JS 文件（原地写回，内容无变化时跳过）
func formatJSFile(path string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file error: %v", err)
	}

	// 使用默认选项调用 jsbeautify 纯 Go 库
	formatted, err := jsbeautify.Format(string(source))
	if err != nil {
		return fmt.Errorf("format error: %v", err)
	}

	if string(source) == formatted {
		return nil
	}

	// 写回时保留原始文件权限
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat file error: %v", err)
	}
	if err := os.WriteFile(path, []byte(formatted), info.Mode()); err != nil {
		return fmt.Errorf("write file error: %v", err)
	}
	return nil
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

	// 第一步：收集所有符合条件的 JS 文件
	logging.Infof("stage 1/2: collect all the js files ...")
	var jsFiles []string
	collectBar := progress.NewSpinner(fmt.Sprintf("collect JS files (%s) ...", mode))

	err = filepath.WalkDir(rootAbsPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			_ = collectBar.Clear()
			logging.Warnf("access path error: %s - %v", path, err)
			errorCount++
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if isJSFile(path) {
			jsFiles = append(jsFiles, path)
			_ = collectBar.Add(1)
		}
		return nil
	})
	_ = collectBar.Finish()

	if err != nil {
		return fmt.Errorf("scan error: %v", err)
	}

	totalFiles := len(jsFiles)
	logging.Infof("found %d JS files to process", totalFiles)

	if totalFiles == 0 {
		return nil
	}

	// 第二步：执行格式化操作
	logging.Infof("stage 2/2: format all the js files (workers: %d) ...", c.Workers)
	formatBar := progress.NewProcessBar(int64(totalFiles), fmt.Sprintf("js format (%s) ...", mode))

	var wg sync.WaitGroup
	sem := make(chan struct{}, c.Workers)
	var mu sync.Mutex

	for i, path := range jsFiles {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(idx int, p string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// 更新计数和进度条
			mu.Lock()
			_ = formatBar.Add(1)
			count++
			currentCount := count
			currentErrorCount := errorCount
			formatBar.Describe(fmt.Sprintf("js format | handle: %d/%d | error: %d", currentCount, totalFiles, currentErrorCount))
			mu.Unlock()

			if c.DryRun {
				return
			}

			// 使用 jsbeautify 纯 Go 库执行格式化
			if err := formatJSFile(p); err != nil {
				mu.Lock()
				_ = formatBar.Clear()
				logging.Errorf("js format error %s: %v", p, err)
				errorCount++
				// 更新进度条以反映新的错误计数
				formatBar.Describe(fmt.Sprintf("js format | handle: %d/%d | error: %d", currentCount, totalFiles, errorCount))
				mu.Unlock()
			} else {
				_ = formatBar.Clear()
				logging.Debugf("js format success: %s", p)
			}
		}(i, path)
	}

	wg.Wait()
	_ = formatBar.Finish()

	logging.Infof("JS format completed: total count: %d , error count: %d", count, errorCount)
	return nil
}
