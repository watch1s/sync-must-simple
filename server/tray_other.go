//go:build !windows

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func PromptUser(title, text string) bool {
	if _, err := exec.LookPath("zenity"); err == nil {
		cmd := exec.Command("zenity", "--question", "--title="+title, "--text="+text)
		return cmd.Run() == nil
	}
	if _, err := exec.LookPath("kdialog"); err == nil {
		cmd := exec.Command("kdialog", "--yesno", text, "--title", title)
		return cmd.Run() == nil
	}
	fmt.Printf("\n[PEER REQUEST] %s\n%s\n(Auto-accepted because no GUI prompt tool found)\n", title, text)
	return true
}

func RunApp(onReady func()) {
	onReady()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("\nShutting down server...")
}
