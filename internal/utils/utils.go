package utils

import (
	"encoding/json"
	"io"
	"os"
	"strings"
)

// SliceUnique 对字符串切片去重，可选忽略大小写、跳过空白项
func SliceUnique(slice []string, ignoreCase, skipEmpty bool) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(slice))

	for _, item := range slice {
		// 处理 skipEmpty：跳过空或纯空白字符串
		if skipEmpty && strings.TrimSpace(item) == "" {
			continue
		}

		// 确定用于去重比较的键（key）
		key := item
		if ignoreCase {
			key = strings.ToLower(item)
		}

		// 如果未见过该 key，则保留原始 item（不是 key！）
		if !seen[key] {
			seen[key] = true
			out = append(out, item) // 保留原始大小写形式
		}
	}

	return out
}

// ToLowerKeys 将字符串切片统一转为小写
func ToLowerKeys(keys []string) []string {
	if len(keys) == 0 {
		return []string{}
	}

	lowerKeys := make([]string, len(keys))
	for i, key := range keys {
		lowerKeys[i] = strings.ToLower(key)
	}
	return lowerKeys
}

// ToJSON 将任意对象序列化为格式化的 JSON 字符串（用于输出）
func ToJSON(v interface{}) string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}

// IsDirEmpty 判断目录是否为空
func IsDirEmpty(dirPath string) (bool, error) {
	f, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	return err == io.EOF, nil
}
