package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"mysh/config"
	"mysh/connection"
	"mysh/executor"
	"mysh/metadata"

	myweb "github.com/eyjian/myweb"
	"github.com/eyjian/myweb/server"
)

var version = "dev"

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	openBrowser := flag.Bool("open", true, "open browser automatically")
	dev := flag.Bool("dev", false, "development mode (proxy to Vite)")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("myweb version %s\n", version)
		os.Exit(0)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config load failed: %s\n", err)
	}

	// Connect to MySQL (from config or default localhost)
	var pool *connection.Pool
	pool, err = connection.New(&cfg.Connection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "MySQL connection failed: %s\n", err)
		fmt.Fprintf(os.Stderr, "Starting without database connection. Use the Connect dialog in browser.\n")
	}

	// Initialize metadata cache
	var meta *metadata.Cache
	if pool != nil {
		meta, _ = metadata.NewCache(pool)
		if meta != nil {
			go func() { _ = meta.Refresh() }()
		}
	}

	// Initialize executor
	var exec *executor.Executor
	if pool != nil {
		exec = executor.New(pool, meta)
	}

	// Create and start server
	srv := server.New(&server.Config{
		Addr:     *addr,
		Dev:      *dev,
		Config:   cfg,
		Pool:     pool,
		Meta:     meta,
		Executor: exec,
		UIFiles:  myweb.UIFiles,
	})

	if *openBrowser && !*dev {
		go openURL(fmt.Sprintf("http://%s", *addr))
	}

	fmt.Printf("myweb listening on http://%s\n", *addr)
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %s\n", err)
		os.Exit(1)
	}
}

func openURL(url string) {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	} else if runtime.GOOS == "windows" {
		name = "start"
	}
	_ = exec.Command(name, url).Start()
}
