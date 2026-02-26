package cleaner

import (
	"context"
	"errors"
	"fmt"
	"github.com/winezer0/xutils/progress"
	"github.com/winezer0/xutils/utils"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/winezer0/xutils/logging"
)

// EmptyCleaner 清理器配置
type EmptyCleaner struct {
	Path   string // 目标路径
	DryRun bool   // 是否模拟运行
}

// NewEmptyCleaner 创建新清理器
func NewEmptyCleaner(path string, dryRun bool) *EmptyCleaner {
	return &EmptyCleaner{
		Path:   path,
		DryRun: dryRun,
	}
}

// RunClean 执行空项清理（两阶段优化版）
func (c *EmptyCleaner) RunClean() error {
	var (
		totalCount   int
		deletedCount int
		errorCount   int
		startTime    = time.Now()
	)

	mode := getMode(c.DryRun)
	fmt.Printf("start (%s) cleaning empty files and dirs...\n", mode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	rootAbsPath, err := filepath.Abs(c.Path)
	if err != nil {
		return fmt.Errorf("invalid dir path: %v", err)
	}

	// === 第一阶段：删除所有空文件 ===
	fmt.Println("stage 1/2: scanning and deleting empty files...")
	var allDirs []string // 收集所有目录（用于第二阶段逆序处理）

	// 初始化第一阶段进度条
	bar1 := progress.NewSpinner(fmt.Sprintf("empty cleaner (%s) ...", mode))

	err = filepath.WalkDir(rootAbsPath, func(path string, d os.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 更新进度条
		_ = bar1.Add(1)
		bar1.Describe(fmt.Sprintf("empty cleaner | del: %d | err: %d", deletedCount, errorCount))

		if err != nil {
			logging.Warnf("access path error: %s - %v", path, err)
			errorCount++
			return nil
		}

		if !isSubPath(path, rootAbsPath) {
			logging.Warnf("path is outside the dirrange: %s not in %s", path, rootAbsPath)
			return filepath.SkipDir
		}

		if d.IsDir() {
			allDirs = append(allDirs, path)
			totalCount++
		} else {
			// 处理文件
			totalCount++
			info, err := d.Info()
			if err != nil {
				logging.Warnf("failed to get file info: %s - %v", path, err)
				errorCount++
				return nil
			}

			if info.Size() == 0 {
				if c.DryRun {
					logging.Infof("[dryrun] would delete empty file: %s", path)
					deletedCount++
				} else {
					if err := os.Remove(path); err != nil {
						logging.Warnf("failed to delete empty file: %s - %v", path, err)
						errorCount++
					} else {
						logging.Infof("successfully deleted empty file: %s", path)
						deletedCount++
					}
				}
				_ = bar1.Clear()
			}
		}

		return nil
	})

	_ = bar1.Finish()

	if err != nil && !errors.Is(err, context.Canceled) {
		logging.Errorf("stage 1 scan error: %v", err)
	}

	// === 第二阶段：从底向上删除空目录 ===
	if err == nil || errors.Is(err, context.Canceled) {
		fmt.Println("stage 2/2: cleaning empty dirs (bottom-up)...")

		// 初始化第二阶段进度条
		bar2 := progress.NewProcessBarByTotalTask(int64(len(allDirs)), fmt.Sprintf("cleaner dir (%s) ...", mode))

		// 逆序处理：确保先处理深层目录
		for i := len(allDirs) - 1; i >= 0; i-- {
			dir := allDirs[i]

			select {
			case <-ctx.Done():
				break
			default:
			}

			// 更新进度条
			_ = bar2.Add(1)

			// 判断目录是否为空（此时已无空文件，只需看是否有子项）
			isEmpty, err := utils.IsDirEmpty(dir)
			if err != nil {
				logging.Warnf("failed to check if dir is empty: %s - %v", dir, err)
				errorCount++
				continue
			}

			if isEmpty {
				totalCount++ // 目录也计入总数
				if c.DryRun {
					logging.Infof("[dryrun] would delete empty dir: %s", dir)
					deletedCount++
				} else {
					if err := os.Remove(dir); err != nil {
						logging.Warnf("failed to delete empty dir: %s - %v", dir, err)
						errorCount++
					} else {
						logging.Infof("successfully deleted empty dir: %s", dir)
						deletedCount++
					}
				}
				_ = bar2.Clear()
			}
		}
		_ = bar2.Finish()
	}

	c.printSummary(totalCount, deletedCount, errorCount, startTime, err)
	return nil
}

// printProgress 输出进度 - 已废弃
// func (c *EmptyCleaner) printProgress(...) {}

// printSummary 输出最终统计
func (c *EmptyCleaner) printSummary(total, deleted, errors int, start time.Time, runErr error) {
	elapsed := time.Since(start).Round(time.Millisecond)
	status := "completed"
	if runErr != nil {
		status = "interrupted"
	}
	fmt.Printf("\n\n=== cleanup complete ===\n")
	fmt.Printf("status: %s\n", status)
	fmt.Printf("total processed: %d items\n", total)
	fmt.Printf("successfully cleaned: %d items\n", deleted)
	fmt.Printf("errors encountered: %d times\n", errors)
	fmt.Printf("total time: %v\n", elapsed)
}
