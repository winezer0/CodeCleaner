package cleaner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewJSFormater 测试构造函数
func TestNewJSFormater(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		dryRun  bool
		workers int
		wantW   int
	}{
		{"normal", ".", true, 4, 4},
		{"zero workers", ".", true, 0, 1},
		{"negative workers", ".", true, -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewJSFormater(tt.path, tt.dryRun, tt.workers)
			if f.Workers != tt.wantW {
				t.Errorf("NewJSFormater().Workers = %v, want %v", f.Workers, tt.wantW)
			}
		})
	}
}

// TestJSFormater_RunClean_DryRun 测试 DryRun 模式下的并发逻辑
// 注意：此测试不检查实际格式化效果，主要测试并发流程是否正常结束
func TestJSFormater_RunClean_DryRun(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "js_cleaner_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建一些模拟 JS 文件
	files := []string{"a.js", "b.js", "c.ts", "d.jsx", "e.txt"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("console.log('hello')"), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
	}

	// 测试用例
	tests := []struct {
		name    string
		workers int
	}{
		{"single worker", 1},
		{"multi workers", 2},
		{"many workers", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaner := NewJSFormater(tmpDir, true, tt.workers)
			if err := cleaner.RunClean(); err != nil {
				t.Errorf("RunClean() error = %v", err)
			}
		})
	}
}

// TestJSFormater_formatJSFile 测试 formatJSFile 真实格式化逻辑（纯 Go 库，无需外部依赖）
func TestJSFormater_formatJSFile(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"simple function", "function run(a,b){return a+b;}"},
		{"object literal", "var o={a:1,b:2};console.log(o);"},
		{"already formatted", "const x = 1;\nconsole.log(x);\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "js_format_test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			path := filepath.Join(tmpDir, "test.js")
			if err := os.WriteFile(path, []byte(tt.source), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			if err := formatJSFile(path); err != nil {
				t.Fatalf("formatJSFile() error = %v", err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read formatted file: %v", err)
			}

			// 格式化后的内容不应出现未展开的压缩代码
			if strings.Contains(string(got), "{return") {
				t.Errorf("file was not formatted, got: %q", string(got))
			}
		})
	}
}

// TestJSFormater_isJSFile 测试文件类型判断逻辑
func TestJSFormater_isJSFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"test.js", true},
		{"TEST.JS", true},
		{"test.ts", true},
		{"test.jsx", true},
		{"test.tsx", true},
		{"test.mjs", true},
		{"test.cjs", true},
		{"test.txt", false},
		{"test.go", false},
		{"js", false},
	}

	for _, tt := range tests {
		if got := isJSFile(tt.filename); got != tt.want {
			t.Errorf("isJSFile(%q) = %v, want %v", tt.filename, got, tt.want)
		}
	}
}
