package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ltaoo/wx_channels_download/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Define CLI flags
	var (
		showVersion = flag.Bool("version", false, "Print version information")
		port        = flag.Int("port", 8888, "Port to listen on for the proxy server") // changed default from 8080 to 8888 since 8080 is always taken on my machine
		outputDir   = flag.String("output", "./downloads", "Directory to save downloaded videos") // default to ./downloads so videos don't clutter cwd
		verbose     = flag.Bool("verbose", false, "Enable verbose logging")
	)

	flag.Parse()

	// Print version and exit if requested
	if *showVersion {
		fmt.Printf("wx_channels_download %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Validate output directory
	if err := ensureDir(*outputDir); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	log.Printf("wx_channels_download %s starting...", version)
	log.Printf("Proxy port: %d", *port)
	log.Printf("Output directory: %s", *outputDir)

	// Start the proxy server
	app := cmd.NewApp(cmd.Config{
		Port:      *port,
		OutputDir: *outputDir,
		Verbose:   *verbose,
	})

	if err := app.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

// ensureDir creates a directory if it does not already exist.
func ensureDir(path string) error {
	if path == "." || path == "" {
		return nil
	}
	return os.MkdirAll(path, 0755)
}
