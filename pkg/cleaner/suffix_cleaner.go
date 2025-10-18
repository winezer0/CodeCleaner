package cleaner

import (
	"codecleaner/internal/config"
	"codecleaner/pkg/logging"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Cleaner 清理选项
type Cleaner struct {
	Path        string        // 根目录路径
	Stored      []string      // 白名单扩展名列表
	Remove      []string      // 移除扩展名列表（兼容旧参数）
	Rmdirs      []string      // 待删除目录名列表（仅匹配目录名，不限制层级）
	DryRun      bool          // 模拟试运行模式（仅统计不删除）
	EnWhite     bool          // 白名单模式（保留Stored列表，删除其他）
	ProgressInt time.Duration // 进度输出时间间隔，默认 10 秒
}

// NewCleaner 创建清理器实例
func NewCleaner(path string, preset config.PresetConfig, enWhite, dryRun bool) *Cleaner {
	return &Cleaner{
		Path:        path,
		Stored:      preset.Stored,
		Remove:      preset.Remove,
		Rmdirs:      preset.RmDirs,
		DryRun:      dryRun,
		EnWhite:     enWhite,
		ProgressInt: 10 * time.Second, // 默认 10 秒输出一次进度
	}
}

// RunClean 执行清理操作
func (c *Cleaner) RunClean() error {
	// 初始化统计变量
	var (
		totalCount   int
		deletedCount int
		errorCount   int
		startTime    = time.Now()
		lastProgress = startTime // 记录上次进度输出时间
	)

	mode := "实际"
	if c.DryRun {
		mode = "模拟"
	}
	fmt.Printf("开始(%s)扫描和删除文件...\n", mode)

	// 上下文管理（处理中断信号）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听中断信号
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

	// 预处理配置参数（统一格式）
	stored := preprocessExtensions(c.Stored)
	remove := preprocessExtensions(c.Remove)
	rmdirs := preprocessDirNames(c.Rmdirs)

	// 验证根路径有效性
	rootAbsPath, err := filepath.Abs(c.Path)
	if err != nil {
		return fmt.Errorf("目录路径无效: %v", err)
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
			logging.Warnf("访问路径失败: %s - %v", path, err)
			errorCount++
			return nil
		}

		// 获取文件信息（WalkDir需要显式获取，减少不必要的系统调用）
		info, err := d.Info()
		if err != nil {
			logging.Warnf("获取信息失败: %s - %v", path, err)
			errorCount++
			return nil
		}

		// 验证当前路径是否在根路径范围内（防止路径穿越）
		if !isSubPath(path, rootAbsPath) {
			logging.Warnf("超出目录范围: %s not in %s", path, rootAbsPath)
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

		// 按时间间隔输出进度（每ProgressInt输出一次）
		now := time.Now()
		if now.Sub(lastProgress) >= c.ProgressInt && totalCount > 0 {
			c.printProgress(totalCount, deletedCount, errorCount, startTime)
			lastProgress = now // 更新上次输出时间
		}

		return nil
	})

	// 输出最终统计（确保最后一次进度被打印）
	c.printSummary(totalCount, deletedCount, errorCount, startTime, err)
	return nil
}

// 处理目录删除逻辑（基于目录名匹配）
func (c *Cleaner) handleRmdirs(path string, rmdirs []string, totalCount, deletedCount, errorCount *int) bool {
	*totalCount++
	currDirName := strings.ToLower(filepath.Base(path))

	// 检查当前目录名是否在目标列表中
	if isDirInList(currDirName, rmdirs) {
		if c.DryRun {
			logging.Infof("[模拟] 将要删除目录: %s", path)
			*deletedCount++
		} else {
			// 实际删除目录（递归删除所有内容）
			if err := os.RemoveAll(path); err != nil {
				logging.Warnf("删除目录失败: %s - %v", path, err)
				*errorCount++
			} else {
				logging.Infof("成功删除目录: %s", path)
				*deletedCount++
			}
		}
		return true
	}
	return false
}

// 处理文件删除逻辑
func (c *Cleaner) handleFiles(path string, storedExts, removeExts []string, totalCount, deletedCount, errorCount *int) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	*totalCount++

	if c.EnWhite {
		// 白名单模式处理 （使用stored列表）
		if len(storedExts) == 0 {
			logging.Warn("白名单模式未配置stored列表，跳过处理")
			return
		}
		if !isExtensionInList(ext, storedExts) {
			c.deleteFile(path, deletedCount, errorCount)
		}
		return
	} else {
		// 普通模式处理（使用Remove列表）
		if len(removeExts) == 0 {
			logging.Warn("黑名单模式未配置remove列表，跳过处理")
			return
		}
		if isExtensionInList(ext, removeExts) {
			c.deleteFile(path, deletedCount, errorCount)
		}
	}
}

// 执行文件删除（根据Try模式决定是否实际删除）
func (c *Cleaner) deleteFile(path string, deletedCount, errorCount *int) {
	if c.DryRun {
		logging.Infof("[模拟] 将要删除文件: %s", path)
		*deletedCount++
		return
	}

	if err := os.Remove(path); err != nil {
		logging.Warnf("删除文件失败: %s - %v", path, err)
		*errorCount++
	} else {
		logging.Infof("成功删除文件: %s", path)
		*deletedCount++
	}
}

// 定期输出进度信息（按时间间隔触发）
func (c *Cleaner) printProgress(totalCount, deletedCount, errorCount int, startTime time.Time) {
	elapsed := time.Since(startTime)
	rate := float64(deletedCount) / elapsed.Seconds()
	fmt.Printf("进度: 总计 %d, 已处理 %d, 错误 %d, 速度 %.2f/秒\n", totalCount, deletedCount, errorCount, rate)
}

// 输出最终统计信息
func (c *Cleaner) printSummary(total, deleted, errorCount int, startTime time.Time, walkErr error) {
	elapsed := time.Since(startTime).Truncate(time.Second)
	mode := "实际"
	if c.DryRun {
		mode = "模拟"
	}
	fmt.Printf("\n%s清理完成! 用时: %v\n", mode, elapsed)
	fmt.Printf("统计: 总处理 %d 项（含目录）, 成功处理 %d 项, 错误 %d 项\n", total, deleted, errorCount)

	if errors.Is(walkErr, context.Canceled) {
		fmt.Println("清理操作已被用户中断")
	} else if walkErr != nil {
		fmt.Printf("清理过程中发生错误: %v\n", walkErr)
	}

}
