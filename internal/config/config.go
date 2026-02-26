package config

import (
	"github.com/winezer0/xutils/utils"
	"os"
	"path/filepath"
)

// Config 配置文件结构
type Config struct {
	Presets map[string]PresetConfig `yaml:"presets"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		// 默认查找顺序
		configPaths := []string{
			".cleaner.yaml",
			filepath.Join(os.Getenv("HOME"), ".config", "cleaner.yaml"),
		}

		for _, path := range configPaths {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		if configPath == "" {
			return &Config{Presets: make(map[string]PresetConfig)}, nil
		}
	}

	var config Config
	err := utils.LoadYAML(configPath, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// GetPreset 获取预设配置
func (c *Config) GetPreset(name string) (*PresetConfig, bool) {
	preset, exists := c.Presets[name]
	return &preset, exists
}
