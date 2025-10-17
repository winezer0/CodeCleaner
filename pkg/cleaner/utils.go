package cleaner

import (
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
