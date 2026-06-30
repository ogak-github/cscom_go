package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"time"

	"yga.my.id/cscom/internal/metrics"
	"yga.my.id/cscom/internal/transport"
)

func generateKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		fmt.Fprintln(os.Stderr, "failed to generate auth key:", err)
		os.Exit(1)
	}
	return hex.EncodeToString(b)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	collector := metrics.NewCollector()
	collector.Refresh()
	collector.Start(ctx, time.Second)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4040"
	}

	webDir := os.Getenv("CSCOM_WEB")
	if webDir == "" {
		webDir = "web"
	}

	authKey := os.Getenv("CSCOM_KEY")
	if authKey == "" {
		authKey = generateKey()
	}

	srv := transport.New(collector, webDir, authKey)

	addr := ":" + port
	fmt.Printf("cscom-serve listening on %s\n", addr)
	fmt.Printf("Dashboard  http://localhost:%s\n", port)
	fmt.Printf("Auth key   %s\n", authKey)
	fmt.Printf("Bearer     Authorization: Bearer %s\n", authKey)
	if err := http.ListenAndServe(addr, srv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
