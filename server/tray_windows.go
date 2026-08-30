//go:build windows

package main

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procLoadIconW           = user32.NewProc("LoadIconW")

	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")

	procShellExecuteW     = shell32.NewProc("ShellExecuteW")
	procShell_NotifyIconW = shell32.NewProc("Shell_NotifyIconW")
	procGetModuleHandleW  = kernel32.NewProc("GetModuleHandleW")
	procMessageBoxW       = user32.NewProc("MessageBoxW")
)

const (
	WM_USER          = 0x0400
	WM_TRAYICON      = WM_USER + 1
	WM_COMMAND       = 0x0111
	WM_DESTROY       = 0x0002
	WM_LBUTTONUP     = 0x0202
	WM_RBUTTONUP     = 0x0205
	WM_RBUTTONDOWN   = 0x0204
	WM_CONTEXTMENU   = 0x007B
	WM_LBUTTONDBLCLK = 0x0203

	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004

	TPM_RETURNCMD   = 0x0100
	TPM_RIGHTBUTTON = 0x0002

	MF_STRING    = 0x00000000
	MF_GRAYED    = 0x00000001
	MF_DISABLED  = 0x00000002
	MF_SEPARATOR = 0x00000800

	IDI_APPLICATION = 32512

	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002

	CMD_HEADER       = 1001
	CMD_OPEN_STATE   = 1002
	CMD_COPY_IP      = 1003
	CMD_EXIT         = 1004
	CMD_MANAGE_PEERS = 1005
)

type NOTIFYICONDATAW struct {
	cbSize            uint32
	_                 uint32
	hWnd              uintptr
	uID               uint32
	uFlags            uint32
	uCallbackMessage  uint32
	_                 uint32
	hIcon             uintptr
	szTip             [128]uint16
	dwState           uint32
	dwStateMask       uint32
	szInfo            [256]uint16
	uTimeoutOrVersion uint32
	szInfoTitle       [64]uint16
	dwInfoFlags       uint32
	guidItem          [16]byte
	hBalloonIcon      uintptr
}

type WNDCLASSEXW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type MSG struct {
	hwnd    uintptr
	message uint32
	_       uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ X, Y int32 }
	_       uint32
}

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
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	textPtr, _ := syscall.UTF16PtrFromString(text)

	ret, _, _ := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(0x00000004|0x00000020|0x00040000), // MB_YESNO | MB_ICONQUESTION | MB_TOPMOST
	)
	return ret == 6 // IDYES
}

// Native Win32 clipboard copy - zero process execution, zero flashing windows
func copyToClipboard(text string) {
	utf16, err := syscall.UTF16FromString(text)
	if err != nil {
		return
	}
	r, _, _ := procOpenClipboard.Call(0)
	if r == 0 {
		return
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()

	size := len(utf16) * 2
	hMem, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE, uintptr(size))
	if hMem == 0 {
		return
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr != 0 {
		src := (*[1 << 30]byte)(unsafe.Pointer(&utf16[0]))[:size:size]
		dst := (*[1 << 30]byte)(unsafe.Pointer(ptr))[:size:size]
		copy(dst, src)
		procGlobalUnlock.Call(hMem)
		procSetClipboardData.Call(CF_UNICODETEXT, hMem)
	}
}

// Native ShellExecuteW to open browser - zero cmd windows
func openBrowser(url string) {
	verbPtr := syscall.StringToUTF16Ptr("open")
	urlPtr := syscall.StringToUTF16Ptr(url)
	procShellExecuteW.Call(0, uintptr(unsafe.Pointer(verbPtr)), uintptr(unsafe.Pointer(urlPtr)), 0, 0, 1 /* SW_SHOWNORMAL */)
}

func wndProc(hWnd, msg, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case WM_TRAYICON:
		msgType := uint32(lParam)
		if msgType == WM_RBUTTONUP || msgType == WM_LBUTTONUP || msgType == WM_RBUTTONDOWN || msgType == WM_CONTEXTMENU {
			showContextMenu(hWnd)
			return 0
		}
		if msgType == WM_LBUTTONDBLCLK {
			openBrowser("http://127.0.0.1:8787/state")
			return 0
		}
	case WM_COMMAND:
		handleCommand(wParam, hWnd)
		return 0
	case WM_DESTROY:
		removeTray(hWnd)
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hWnd, msg, wParam, lParam)
	return r
}

func showContextMenu(hWnd uintptr) {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	localIP := getLocalIP()
	serverAddr := fmt.Sprintf("%s:8787", localIP)

	headerText := syscall.StringToUTF16Ptr(fmt.Sprintf("sync-must-simple (%s)", serverAddr))
	openText := syscall.StringToUTF16Ptr("Open Status in Browser")
	manageText := syscall.StringToUTF16Ptr("Manage Peers")
	copyText := syscall.StringToUTF16Ptr(fmt.Sprintf("Copy Server Address (%s:8788)", localIP))
	exitText := syscall.StringToUTF16Ptr("Quit Server")

	procAppendMenuW.Call(hMenu, MF_STRING|MF_GRAYED, CMD_HEADER, uintptr(unsafe.Pointer(headerText)))
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	procAppendMenuW.Call(hMenu, MF_STRING, CMD_OPEN_STATE, uintptr(unsafe.Pointer(openText)))
	procAppendMenuW.Call(hMenu, MF_STRING, CMD_MANAGE_PEERS, uintptr(unsafe.Pointer(manageText)))
	procAppendMenuW.Call(hMenu, MF_STRING, CMD_COPY_IP, uintptr(unsafe.Pointer(copyText)))
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	procAppendMenuW.Call(hMenu, MF_STRING, CMD_EXIT, uintptr(unsafe.Pointer(exitText)))

	var pt struct{ X, Y int32 }
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	procSetForegroundWindow.Call(hWnd)

	// TPM_RETURNCMD directly returns the clicked menu ID synchronously
	cmd, _, _ := procTrackPopupMenu.Call(
		hMenu,
		TPM_RETURNCMD|TPM_RIGHTBUTTON,
		uintptr(pt.X),
		uintptr(pt.Y),
		0,
		hWnd,
		0,
	)

	if cmd != 0 {
		handleCommand(cmd, hWnd)
	}
}

func handleCommand(cmd uintptr, hWnd uintptr) {
	switch cmd {
	case CMD_OPEN_STATE:
		openBrowser("http://127.0.0.1:8787/state")
	case CMD_MANAGE_PEERS:
		openBrowser("http://127.0.0.1:8788/manage")
	case CMD_COPY_IP:
		localIP := getLocalIP()
		copyToClipboard(fmt.Sprintf("%s:8788", localIP))
	case CMD_EXIT:
		removeTray(hWnd)
		procDestroyWindow.Call(hWnd)
		os.Exit(0)
	}
}

func createTrayIcon(hWnd uintptr) {
	hIcon, _, _ := procLoadIconW.Call(0, uintptr(IDI_APPLICATION))

	var nid NOTIFYICONDATAW
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = hWnd
	nid.uID = 1
	nid.uFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	nid.uCallbackMessage = WM_TRAYICON
	nid.hIcon = hIcon

	tip := "sync-must-simple server (Port: 8787)"
	utf16Tip, _ := syscall.UTF16FromString(tip)
	copy(nid.szTip[:], utf16Tip)

	procShell_NotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))
}

func removeTray(hWnd uintptr) {
	var nid NOTIFYICONDATAW
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = hWnd
	nid.uID = 1
	procShell_NotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
}

func RunApp(onReady func()) {
	// Lock this goroutine to its current OS thread for the Win32 message loop
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	className := syscall.StringToUTF16Ptr("SyncMustSimpleTrayClass")

	var wc WNDCLASSEXW
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.lpfnWndProc = syscall.NewCallback(wndProc)
	wc.hInstance = hInstance
	wc.lpszClassName = className

	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	windowName := syscall.StringToUTF16Ptr("sync-must-simple")
	hWnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		0,
		0, 0, 0, 0,
		0, 0, hInstance, 0,
	)

	createTrayIcon(hWnd)

	// Start server in background
	onReady()

	// Windows Message Loop
	var msg MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
