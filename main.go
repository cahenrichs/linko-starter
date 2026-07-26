package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"boot.dev/linko/internal/store"
)

var logger = log.New(os.Stderr, "DEBUG: ", log.LstdFlags)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	httpPort := flag.Int("port", 8899, "port to listen on")
	dataDir := flag.String("data", "./data", "directory to store data")
	flag.Parse()

	status := run(ctx, cancel, *httpPort, *dataDir)
	cancel()
	os.Exit(status)
}

func run(ctx context.Context, cancel context.CancelFunc, httpPort int, dataDir string) int {
	//standard logger
	standard := log.New(os.Stderr, "DEBUG: ", log.LstdFlags)

	//access log creation
	create, err := os.Create("linko.access.log")
	if err != nil {
		standard.Printf("failed to create store: %v\n", err)
		return 1
	}
	access := log.New(create, "INFO: ", log.LstdFlags)

	st, err := store.New(dataDir, standard)
	if err != nil {
		logger.Printf("failed to create store: %v\n", err)
		return 1
	}
	s := newServer(*st, httpPort, cancel, access)
	var serverErr error
	go func() {
		serverErr = s.start()
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.shutdown(shutdownCtx); err != nil {
		access.Printf("failed to shutdown server: %v\n", err)
		return 1
	}
	if serverErr != nil {
		access.Printf("server error: %v\n", serverErr)
		return 1
	}
	return 0
}
