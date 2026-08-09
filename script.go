package main

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	MOD_CONTROL = 0x0002
	MOD_SHIFT   = 0x0004
	HOTKEY_ID   = 1
	HOTKEY_CLOSE_ID = 2
	VK_C        = 0x43
	VK_DELETE   = 0x2E

	WM_HOTKEY = 0x0312

	SW_HIDE = 0
	SW_SHOW = 5

	GWL_EXSTYLE      = 0xFFFFFFFFFFFFFFEC
	WS_EX_TOOLWINDOW = 0x00000080
	WS_EX_APPWINDOW  = 0x00040000

	SWP_NOMOVE       = 0x0002
	SWP_NOSIZE       = 0x0001
	SWP_NOZORDER     = 0x0004
	SWP_FRAMECHANGED = 0x0020
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")

	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procGetMessage       = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessage  = user32.NewProc("DispatchMessageW")

	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procShowWindow          = user32.NewProc("ShowWindow")
	procGetWindowLongPtr    = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtr    = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
)

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct {
		X int32
		Y int32
	}
}

var (
	hiddenHWND uintptr
	isHidden   bool
)

func getWindowExStyle(hwnd uintptr) uintptr {
	val, _, _ := procGetWindowLongPtr.Call(hwnd, GWL_EXSTYLE)
	return val
}

func setWindowExStyle(hwnd, style uintptr) {
	procSetWindowLongPtr.Call(hwnd, GWL_EXSTYLE, style)
}

func hideWindow(hwnd uintptr) {
	style := getWindowExStyle(hwnd)
	newStyle := style | WS_EX_TOOLWINDOW
	newStyle &^= WS_EX_APPWINDOW
	setWindowExStyle(hwnd, newStyle)

	procSetWindowPos.Call(
		hwnd,
		0,
		0, 0, 0, 0,
		SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER|SWP_FRAMECHANGED,
	)

	procShowWindow.Call(hwnd, SW_HIDE)
}

func restoreWindow(hwnd uintptr) {
	style := getWindowExStyle(hwnd)
	newStyle := style | WS_EX_APPWINDOW
	newStyle &^= WS_EX_TOOLWINDOW
	setWindowExStyle(hwnd, newStyle)

	procSetWindowPos.Call(
		hwnd,
		0,
		0, 0, 0, 0,
		SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER|SWP_FRAMECHANGED,
	)

	procShowWindow.Call(hwnd, SW_SHOW)
}

func toggle() {
	if !isHidden {
		hwnd, _, _ := procGetForegroundWindow.Call()
		if hwnd == 0 {
			return
		}
		hiddenHWND = hwnd
		hideWindow(hwnd)
		isHidden = true
	} else {
		restoreWindow(hiddenHWND)
		isHidden = false
	}
}

func closeApp() {
	procPostQuitMessage.Call(0)
}

func messageLoop() {
	var msg MSG
	for {
		ret, _, _ := procGetMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0,
			0,
			0,
		)
		if ret == 0 {
			break
		}

		if msg.Message == WM_HOTKEY {
			switch msg.WParam {
			case HOTKEY_ID:
				toggle()
			case HOTKEY_CLOSE_ID:
				closeApp()
			}
		}

		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func main() {
	runtime.LockOSThread()

	procRegisterHotKey.Call(
		0,
		HOTKEY_ID,
		MOD_CONTROL|MOD_SHIFT,
		uintptr(VK_C),
	)
	defer procUnregisterHotKey.Call(0, HOTKEY_ID)

	procRegisterHotKey.Call(
		0,
		HOTKEY_CLOSE_ID,
		MOD_CONTROL|MOD_SHIFT,
		uintptr(VK_DELETE),
	)
	defer procUnregisterHotKey.Call(0, HOTKEY_CLOSE_ID)

	messageLoop()
}
