package cleaner

import (
	"os"
	"path/filepath"
	"strings"
)

// 检查扩展名是否在目标列表中
func isExtensionInList(ext string, extensions []string) bool {
	for _, e := range extensions {
		if e == "none" && ext == "" {
			return true
		}
		if ext == e {
			return true
		}
	}
	return false
}

// 检查Dir是否在目标列表中
func isDirInList(dir string, dirs []string) bool {
	for _, e := range dirs {
		if dir == e {
			return true
		}
	}
	return false
}

// 预处理扩展名（统一转为小写，去除前缀点）
func preprocessExtensions(exts []string) []string {
	result := make([]string, 0, len(exts))
	for _, e := range exts {
		cleaned := strings.ToLower(strings.TrimPrefix(e, "."))
		result = append(result, cleaned)
	}
	return result
}

// 检查path是否为parent的子路径（包含自身）
func isSubPath(path, parent string) bool {
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

// 预处理目录名（统一格式，可选大小写转换）
func preprocessDirNames(names []string) []string {
	result := make([]string, 0, len(names))
	for _, name := range names {
		cleaned := strings.TrimSpace(strings.ToLower(name))
		if cleaned != "" {
			result = append(result, cleaned)
		}
	}
	return result
}

// IsDirEmpty 判断指定路径的目录是否为空
func IsDirEmpty(dirPath string) bool {
	// 打开目录
	dir, err := os.Open(dirPath)
	if err != nil {
		return false
	}
	defer dir.Close() // 确保目录句柄关闭

	// 读取目录项，最多读取1个（只要有内容就不是空目录）
	entries, err := dir.Readdir(1)
	if err != nil {
		return false
	}

	// 目录项长度为0表示空目录
	return len(entries) == 0
}

// IsFileEmpty 判断指定路径的文件是否为空
func IsFileEmpty(filePath string) bool {
	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	// 验证路径是否为文件（排除目录）
	if fileInfo.IsDir() {
		return false
	}

	// 文件大小为0表示空文件
	return fileInfo.Size() == 0
}
