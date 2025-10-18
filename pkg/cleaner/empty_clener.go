package cleaner

import (
	"codecleaner/pkg/logging"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// EmptyCleaner 清理器配置
type EmptyCleaner struct {
	Path        string        // 目标路径
	DryRun      bool          // 是否模拟运行
	ProgressInt time.Duration // 进度更新间隔
}

// NewEmptyCleaner 创建新清理器
func NewEmptyCleaner(path string, dryRun bool) *EmptyCleaner {
	return &EmptyCleaner{
		Path:        path,
		DryRun:      dryRun,
		ProgressInt: 2 * time.Second,
	}
}

// RunClean 执行空项清理（两阶段优化版）
func (c *EmptyCleaner) RunClean() error {
	var (
		totalCount   int
		deletedCount int
		errorCount   int
		startTime    = time.Now()
		lastProgress = startTime
	)

	mode := "实际"
	if c.DryRun {
		mode = "模拟"
	}
	fmt.Printf("开始(%s)清理空文件和空目录...\n", mode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			fmt.Println("\n收到中断信号，正在停止操作...")
			cancel()
		case <-ctx.Done():
		}
	}()

	rootAbsPath, err := filepath.Abs(c.Path)
	if err != nil {
		return fmt.Errorf("目录路径无效: %v", err)
	}

	// === 第一阶段：删除所有空文件 ===
	fmt.Println("阶段 1/2: 扫描并删除空文件...")
	var allDirs []string // 收集所有目录（用于第二阶段逆序处理）

	err = filepath.WalkDir(rootAbsPath, func(path string, d os.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			logging.Warnf("访问路径失败: %s - %v", path, err)
			errorCount++
			return nil
		}

		if !isSubPath(path, rootAbsPath) {
			logging.Warnf("超出目录范围: %s not in %s", path, rootAbsPath)
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
				logging.Warnf("获取文件信息失败: %s - %v", path, err)
				errorCount++
				return nil
			}

			if info.Size() == 0 {
				if c.DryRun {
					logging.Infof("[模拟] 将要删除空文件: %s", path)
					deletedCount++
				} else {
					if err := os.Remove(path); err != nil {
						logging.Warnf("删除空文件失败: %s - %v", path, err)
						errorCount++
					} else {
						logging.Infof("成功删除空文件: %s", path)
						deletedCount++
					}
				}
			}
		}

		// 定期输出进度
		now := time.Now()
		if now.Sub(lastProgress) >= c.ProgressInt && totalCount > 0 {
			c.printProgress(totalCount, deletedCount, errorCount, startTime)
			lastProgress = now
		}

		return nil
	})

	if err != nil && !errors.Is(err, context.Canceled) {
		logging.Errorf("第一阶段扫描出错: %v", err)
	}

	// === 第二阶段：从底向上删除空目录 ===
	if err == nil || errors.Is(err, context.Canceled) {
		fmt.Println("阶段 2/2: 清理空目录（从底向上）...")

		// 逆序处理：确保先处理深层目录
		for i := len(allDirs) - 1; i >= 0; i-- {
			dir := allDirs[i]

			select {
			case <-ctx.Done():
				break
			default:
			}

			// 判断目录是否为空（此时已无空文件，只需看是否有子项）
			isEmpty, err := isDirEmpty(dir)
			if err != nil {
				logging.Warnf("检查目录是否为空失败: %s - %v", dir, err)
				errorCount++
				continue
			}

			if isEmpty {
				totalCount++ // 目录也计入总数
				if c.DryRun {
					logging.Infof("[模拟] 将要删除空目录: %s", dir)
					deletedCount++
				} else {
					if err := os.Remove(dir); err != nil {
						logging.Warnf("删除空目录失败: %s - %v", dir, err)
						errorCount++
					} else {
						logging.Infof("成功删除空目录: %s", dir)
						deletedCount++
					}
				}
			}
		}
	}

	c.printSummary(totalCount, deletedCount, errorCount, startTime, err)
	return nil
}

// isDirEmpty 判断目录是否为空（无任何子项）
func isDirEmpty(dirPath string) (bool, error) {
	f, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	return err == io.EOF, nil
}

// printProgress 输出进度
func (c *EmptyCleaner) printProgress(total, deleted, errors int, start time.Time) {
	elapsed := time.Since(start).Round(time.Millisecond)
	rate := float64(total) / elapsed.Seconds()
	fmt.Printf("扫描中: 总计 %d | 删除 %d | 错误 %d | 速度 %.1f/s | 用时 %v\r",
		total, deleted, errors, rate, elapsed)
}

// printSummary 输出最终统计
func (c *EmptyCleaner) printSummary(total, deleted, errors int, start time.Time, runErr error) {
	elapsed := time.Since(start).Round(time.Millisecond)
	status := "完成"
	if runErr != nil {
		status = "中断"
	}
	fmt.Printf("\n\n=== 清理完成 ===\n")
	fmt.Printf("状态: %s\n", status)
	fmt.Printf("总计处理: %d 项\n", total)
	fmt.Printf("成功清理: %d 项\n", deleted)
	fmt.Printf("遇到错误: %d 次\n", errors)
	fmt.Printf("总耗时: %v\n", elapsed)
}
