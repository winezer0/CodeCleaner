package main

import (
	"codecleaner/pkg/logging"
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
)

// Options command line options
type Options struct {
	Path string `short:"p" long:"path" description:"Code directory path to scan and clear (default: null)"`

	// Log configuration
	LogFile       string `long:"lf" description:"Log file path (default: null)" `
	LogLevel      string `long:"ll" description:"Log level (debug/info/warn/error)" default:"info"`
	ConsoleFormat string `long:"cf" description:"Console log format (T L C M F combination or off|null to disable)" default:"M"`
}

func main() {
	var opts Options

	parser := flags.NewParser(&opts, flags.Default)
	parser.Usage = "[OPTIONS]"

	// Custom help information
	parser.LongDescription = `Code line count tool`

	if _, err := parser.Parse(); err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && errors.Is(flagsErr.Type, flags.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "cmd options parsing error: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logCfg := logging.NewLogConfig(opts.LogLevel, opts.LogFile, opts.ConsoleFormat)
	if err := logging.InitLogger(logCfg); err != nil {
		// Cannot use logging here as it's not initialized yet
		fmt.Printf("Failed to initialize the logger: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync()
}
