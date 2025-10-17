package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

// PresetConfig 预设配置结构
type PresetConfig struct {
	Description string   `yaml:"description"`
	Stored      []string `yaml:"stored"`
	Remove      []string `yaml:"remove"`
	RmDirs      []string `yaml:"rmdirs"`
}

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

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
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