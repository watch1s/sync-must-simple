//go:build !systray

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func RunApp(onReady func()) {
	fmt.Println("Running WITHOUT systray support (no -tags systray provided).")
	onReady()
	
	// Block forever until sigterm
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("Shutting down...")
}
