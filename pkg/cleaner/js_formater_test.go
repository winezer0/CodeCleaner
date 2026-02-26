package cleaner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewJsCleaner(t *testing.T) {
	path := "."
	dryRun := true
	cleaner := NewJsCleaner(path, dryRun)

	if cleaner.Path != path {
		t.Errorf("Expected path %s, got %s", path, cleaner.Path)
	}
	if cleaner.DryRun != dryRun {
		t.Errorf("Expected DryRun %v, got %v", dryRun, cleaner.DryRun)
	}
}

func TestJsCleaner_RunClean(t *testing.T) {
	// Check if js-beautify is installed
	if err := exec.Command("js-beautify", "--version").Run(); err != nil {
		t.Skip("js-beautify not installed, skipping integration test")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "js_cleaner_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a messy JS file
	jsContent := `var a=1;if(a){console.log("hello");}`
	jsFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(jsFile, []byte(jsContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run cleaner
	cleaner := NewJsCleaner(tmpDir, false)
	if err := cleaner.RunClean(); err != nil {
		t.Fatalf("RunClean failed: %v", err)
	}

	// Read file and check if formatted
	formattedContent, err := os.ReadFile(jsFile)
	if err != nil {
		t.Fatalf("Failed to read formatted file: %v", err)
	}

	// js-beautify usually adds spaces and newlines
	if string(formattedContent) == jsContent {
		t.Errorf("File was not formatted. Content: %s", string(formattedContent))
	}
	
	// Basic check for formatting (multiline)
	if !strings.Contains(string(formattedContent), "\n") {
		t.Errorf("File should contain newlines after formatting")
	}
}
