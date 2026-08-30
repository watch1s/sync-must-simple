//go:build systray

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
)

// A tiny transparent 1x1 PNG icon
var iconData = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

func RunApp(onReady func()) {
	systray.Run(func() {
		onReady()
		setupTrayMenu()
	}, func() {
		fmt.Println("Shutting down...")
		os.Exit(0)
	})
}

func setupTrayMenu() {
	systray.SetIcon(iconData)
	systray.SetTitle("Sync")
	systray.SetTooltip("sync-must-simple server")

	mHeader := systray.AddMenuItem("sync-must-simple — port: 8787", "Status")
	mHeader.Disable() // Just a header

	systray.AddSeparator()
	mOpen := systray.AddMenuItem("Aç", "Tarayıcıda durumu aç")
	mQuit := systray.AddMenuItem("Kapat", "Sunucuyu kapat")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openBrowser("http://127.0.0.1:8787/state")
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	}
	if err != nil {
		fmt.Println("Error opening browser:", err)
	}
}
