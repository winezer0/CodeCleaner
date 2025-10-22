package cmdutils

import "strings"

// ParseCommaStrToList 解析逗号分隔字符串为列表
func ParseCommaStrToList(CommaStr string, toLower bool) []string {
	var result []string
	if CommaStr != "" {
		strList := strings.Split(CommaStr, ",")
		for _, str := range strList {
			str = strings.TrimSpace(str)
			if str == "" {
				continue
			}
			// Convert to lowercase for consistency
			if toLower {
				str = strings.ToLower(str)
			}
			result = append(result, str)
		}
	}
	return result
}
