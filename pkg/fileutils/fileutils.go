package fileutils

import (
	"fmt"
	"os"
	"path/filepath"
)

// MakeDirs 创建指定路径的所有必要目录（递归创建）
func MakeDirs(path string, isFile bool) error {
	if isFile {
		path, _ = GetFileDirectory(path)
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create the directory: %w", err)
	}
	return nil
}

// GetFileDirectory 返回给定文件路径的目录部分
func GetFileDirectory(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("文件路径不能为空")
	}

	dir := filepath.Dir(filePath)
	return dir, nil
}

// IsEmptyFile 检查文件是否为空或不存在
func IsEmptyFile(filename string) bool {
	// Get file info
	fileInfo, err := os.Stat(filename)
	if os.IsNotExist(err) || fileInfo.Size() == 0 {
		return true
	}
	return false
}

// WriteAny 将任意数据写入文本文件
func WriteAny(filePath string, data interface{}) error {
	// 将任意数据转换为字符串形式
	content := fmt.Sprintf("%+v", data)

	// 写入文件
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("file write failed: %w", err)
	}

	return nil
}
