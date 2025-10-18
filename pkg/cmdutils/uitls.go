package cmdutils

// ListUnique list去重
func ListUnique(input []string) []string {
	seen := make(map[string]struct{}) // 用作集合
	var result []string

	for _, s := range input {
		if _, exists := seen[s]; !exists {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}

	return result
}
