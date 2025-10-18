package config

// PresetConfig 预设配置结构
type PresetConfig struct {
	Description string   `yaml:"description"`
	Stored      []string `yaml:"stored"`
	Remove      []string `yaml:"remove"`
	RmDirs      []string `yaml:"rmdirs"`
}

// NewPresetConfig  带参数的构造函数
func NewPresetConfig(description string, stored, remove, rmdirs []string) *PresetConfig {
	return &PresetConfig{
		Description: description,
		Stored:      copyStrSlice(stored),
		Remove:      copyStrSlice(remove),
		RmDirs:      copyStrSlice(rmdirs),
	}
}

// copyStrSlice safely copies a string slice, handling nil case.
func copyStrSlice(src []string) []string {
	if src == nil {
		return []string{}
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}
