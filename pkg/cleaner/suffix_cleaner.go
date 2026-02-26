package cleaner

import (
	"codecleaner/internal/config"
	"context"
	"errors"
	"fmt"
	"github.com/winezer0/xutils/progress"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/winezer0/xutils/logging"
)

// SuffixCleaner 清理选项
type SuffixCleaner struct {
	Path    string   // 根目录路径
	Stored  []string // 白名单扩展名列表
	Remove  []string // 移除扩展名列表（兼容旧参数）
	Rmdirs  []string // 待删除目录名列表（仅匹配目录名，不限制层级）
	DryRun  bool     // 模拟试运行模式（仅统计不删除）
	EnWhite bool     // 白名单模式（保留Stored列表，删除其他）

	bar *progressbar.ProgressBar
}

// NewSuffixCleaner 创建清理器实例
func NewSuffixCleaner(path string, preset config.PresetConfig, enWhite, dryRun bool) *SuffixCleaner {
	bar := progress.NewSpinner(fmt.Sprintf("suffix clean (%s) ...", getMode(dryRun)))
	return &SuffixCleaner{
		Path:    path,
		Stored:  preset.Stored,
		Remove:  preset.Remove,
		Rmdirs:  preset.RmDirs,
		DryRun:  dryRun,
		EnWhite: enWhite,
		bar:     bar,
	}
}

// RunClean 执行清理操作
func (c *SuffixCleaner) RunClean() error {
	// 初始化统计变量
	var (
		totalCount   int
		deletedCount int
		errorCount   int
		startTime    = time.Now()
	)

	mode := getMode(c.DryRun)
	logging.Infof("start (%s) scanning and deleting files...\n", mode)

	// 上下文管理（处理中断信号）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			fmt.Println("\nreceived interrupt signal, stopping operation...")
			cancel()
		case <-ctx.Done():
		}
	}()

	// 预处理配置参数（统一格式）
	stored := preprocessExtensions(c.Stored)
	remove := preprocessExtensions(c.Remove)
	rmdirs := preprocessDirNames(c.Rmdirs)

	// 验证根路径有效性
	rootAbsPath, err := filepath.Abs(c.Path)
	if err != nil {
		return fmt.Errorf("invalid dir path: %v", err)
	}

	// 替换为 filepath.WalkDir 提升性能
	err = filepath.WalkDir(rootAbsPath, func(path string, d os.DirEntry, err error) error {
		// 检查中断信号
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 处理文件访问错误
		if err != nil {
			logging.Warnf("failed to access path: %s - %v", path, err)
			errorCount++
			return nil
		}

		// 更新进度条
		_ = c.bar.Add(1)
		c.bar.Describe(fmt.Sprintf("suffix clean ... | deleted: %d | errors: %d", deletedCount, errorCount))

		// 获取文件信息（WalkDir需要显式获取，减少不必要的系统调用）
		info, err := d.Info()
		if err != nil {
			logging.Warnf("failed to get info: %s - %v", path, err)
			errorCount++
			return nil
		}

		// 验证当前路径是否在根路径范围内（防止路径穿越）
		if !isSubPath(path, rootAbsPath) {
			logging.Warnf("path out of range: %s not in %s", path, rootAbsPath)
			return filepath.SkipDir
		}

		if info.IsDir() {
			// 优先处理目录删除
			if len(rmdirs) > 0 {
				if c.handleRmdirs(path, rmdirs, &totalCount, &deletedCount, &errorCount) {
					return filepath.SkipDir
				}
			}
		} else {
			// 非目录文件处理
			if len(stored) > 0 || len(remove) > 0 {
				c.handleFiles(path, stored, remove, &totalCount, &deletedCount, &errorCount)
			}
		}

		return nil
	})

	// 输出最终统计（确保最后一次进度被打印）
	_ = c.bar.Finish() // Ensure bar is finished before summary
	c.printSummary(totalCount, deletedCount, errorCount, startTime, err)
	return nil
}

// 处理目录删除逻辑（基于目录名匹配）
func (c *SuffixCleaner) handleRmdirs(path string, rmdirs []string, totalCount, deletedCount, errorCount *int) bool {
	*totalCount++
	currDirName := strings.ToLower(filepath.Base(path))
	_ = c.bar.Clear()

	// 检查当前目录名是否在目标列表中
	if isDirInList(currDirName, rmdirs) {
		if c.DryRun {
			logging.Infof("[dryrun] would delete dir: %s", path)
			*deletedCount++
		} else {
			// 实际删除目录（递归删除所有内容）
			if err := os.RemoveAll(path); err != nil {
				logging.Warnf("failed to delete dir: %s - %v", path, err)
				*errorCount++
			} else {
				logging.Infof("successfully deleted dir: %s", path)
				*deletedCount++
			}
		}
		return true
	}
	return false
}

// 处理文件删除逻辑
func (c *SuffixCleaner) handleFiles(path string, storedExts, removeExts []string, totalCount, deletedCount, errorCount *int) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	*totalCount++
	_ = c.bar.Clear()

	if c.EnWhite {
		// 白名单模式处理 （使用stored列表）
		if len(storedExts) == 0 {
			logging.Warn("whitelist mode: no stored list configured, skipping")
			return
		}

		if !isExtensionInList(ext, storedExts) {
			c.deleteFile(path, deletedCount, errorCount)
		}

	} else {
		// 普通模式处理（使用Remove列表）
		if len(removeExts) == 0 {
			logging.Warn("blacklist mode: no remove list configured, skipping")
			return
		}

		if isExtensionInList(ext, removeExts) {
			c.deleteFile(path, deletedCount, errorCount)
		}
	}
}

// 执行文件删除（根据Try模式决定是否实际删除）
func (c *SuffixCleaner) deleteFile(path string, deletedCount, errorCount *int) {
	if c.DryRun {
		logging.Infof("[dryrun] would delete file: %s", path)
		*deletedCount++
		return
	}

	if err := os.Remove(path); err != nil {
		logging.Warnf("failed to delete file: %s - %v", path, err)
		*errorCount++
	} else {
		logging.Infof("successfully deleted file: %s", path)
		*deletedCount++
	}
}

// 输出最终统计信息
func (c *SuffixCleaner) printSummary(total, deleted, errorCount int, startTime time.Time, walkErr error) {
	elapsed := time.Since(startTime).Truncate(time.Second)
	mode := getMode(c.DryRun)
	fmt.Printf("\n%s cleanup complete! time: %v\n", mode, elapsed)
	fmt.Printf("stats: total processed %d items (incl dirs), successfully processed %d items, errors %d\n", total, deleted, errorCount)

	if errors.Is(walkErr, context.Canceled) {
		fmt.Println("cleanup operation interrupted by user")
	} else if walkErr != nil {
		fmt.Printf("error during cleanup: %v\n", walkErr)
	}

}
