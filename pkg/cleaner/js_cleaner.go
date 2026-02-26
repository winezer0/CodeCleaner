package cleaner

import (
	"codecleaner/pkg/logging"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// JsCleaner JS格式化清理器
type JsCleaner struct {
	Path   string // 根目录路径
	DryRun bool   // 模拟模式
}

// NewJsCleaner 创建JS清理器实例
func NewJsCleaner(path string, dryRun bool) *JsCleaner {
	return &JsCleaner{
		Path:   path,
		DryRun: dryRun,
	}
}

// RunClean 执行JS格式化操作
func (c *JsCleaner) RunClean() error {
	logging.Infof("开始(%s) JS代码格式化...", func() string {
		if c.DryRun {
			return "模拟"
		}
		return "实际"
	}())

	// 验证根路径
	rootAbsPath, err := filepath.Abs(c.Path)
	if err != nil {
		return fmt.Errorf("目录路径无效: %v", err)
	}

	var count int
	var errorCount int

	err = filepath.WalkDir(rootAbsPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			logging.Warnf("访问路径失败: %s - %v", path, err)
			errorCount++
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// 仅处理 .js 文件
		if strings.ToLower(filepath.Ext(path)) == ".js" {
			count++
			if c.DryRun {
				logging.Infof("[DryRun] Would format: %s", path)
				return nil
			}

			// 执行 js-beautify
			cmd := exec.Command("js-beautify", "-r", path)
			output, err := cmd.CombinedOutput()
			if err != nil {
				logging.Errorf("格式化失败 %s: %v, output: %s", path, err, string(output))
				errorCount++
			} else {
				logging.Debugf("已格式化: %s", path)
			}
		}
		return nil
	})

	logging.Infof("JS格式化完成: 总计处理 %d 个文件, 失败 %d 个", count, errorCount)
	return err
}
