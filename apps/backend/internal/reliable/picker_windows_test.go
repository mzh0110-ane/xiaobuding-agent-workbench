package reliable

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

var pickerUser32 = windows.NewLazySystemDLL("user32.dll")

// This opt-in test opens a real dialog on the interactive Windows desktop.
func TestNativeProjectPickerVisibleAndCancellation(t *testing.T) {
	if os.Getenv("XIAOBUDING_NATIVE_PICKER_TEST") != "1" {
		t.Skip("requires the interactive Windows desktop")
	}
	for _, closeWith := range []string{"select_folder", "cancel_button", "request_cancel"} {
		t.Run(closeWith, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			t.Cleanup(cancel)
			done := make(chan struct{})
			var path string
			var err error
			go func() { path, err = pickProject(ctx); close(done) }()
			t.Cleanup(func() {
				cancel()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Error("picker process did not exit")
				}
			})
			var hwnd uintptr
			require.Eventually(t, func() bool {
				hwnd = ownedPickerDialog()
				return hwnd != 0
			}, 10*time.Second, 25*time.Millisecond, "no visible folder picker opened")
			style, _, _ := pickerUser32.NewProc("GetWindowLongPtrW").Call(hwnd, ^uintptr(19))
			require.NotZero(t, style&8, "folder picker can be obscured by the browser (WS_EX_TOPMOST missing)")
			folder := filepath.Join(t.TempDir(), "项目 中文")
			require.NoError(t, os.Mkdir(folder, 0700))
			switch closeWith {
			case "select_folder":
				selectNativePickerFolder(t, hwnd, folder)
			case "cancel_button":
				_, _, _ = pickerUser32.NewProc("PostMessageW").Call(hwnd, 0x111, 2, 0)
			default:
				cancel()
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("folder picker did not settle after cancellation")
			}
			if closeWith == "select_folder" {
				require.Equal(t, folder, path)
			} else {
				require.Empty(t, path, "cancellation must not select a project")
			}
			if closeWith != "request_cancel" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Eventually(t, func() bool { return ownedPickerDialog() == 0 }, time.Second, 25*time.Millisecond)
		})
	}
}

func selectNativePickerFolder(t *testing.T, hwnd uintptr, folder string) {
	t.Helper()
	var pid uint32
	_, _, _ = pickerUser32.NewProc("GetWindowThreadProcessId").Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	require.True(t, pickerChildProcesses()[pid], "only control a picker owned by this test")
	process, err := windows.OpenProcess(windows.PROCESS_VM_OPERATION|windows.PROCESS_VM_WRITE, false, pid)
	require.NoError(t, err)
	t.Cleanup(func() { _ = windows.CloseHandle(process) })
	text, err := windows.UTF16FromString(folder)
	require.NoError(t, err)
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	size := uintptr(len(text) * 2)
	remote, _, allocErr := kernel.NewProc("VirtualAllocEx").Call(uintptr(process), 0, size, windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	require.NotZero(t, remote, "%v", allocErr)
	defer func() {
		_, _, _ = kernel.NewProc("VirtualFreeEx").Call(uintptr(process), remote, 0, windows.MEM_RELEASE)
	}()
	require.NoError(t, windows.WriteProcessMemory(process, remote, (*byte)(unsafe.Pointer(&text[0])), size, nil))
	// BFFM_SETSELECTIONW is a private message, so its path lives in the owned UI process.
	_, _, _ = pickerUser32.NewProc("SendMessageTimeoutW").Call(hwnd, 0x467, 1, remote, 2, 2000, 0)
	_, _, _ = pickerUser32.NewProc("PostMessageW").Call(hwnd, 0x111, 1, 0)
}

func ownedPickerDialog() uintptr {
	children := pickerChildProcesses()
	var found uintptr
	callback := windows.NewCallback(func(hwnd, _ uintptr) uintptr {
		var pid uint32
		_, _, _ = pickerUser32.NewProc("GetWindowThreadProcessId").Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		visible, _, _ := pickerUser32.NewProc("IsWindowVisible").Call(hwnd)
		class := make([]uint16, 128)
		_, _, _ = pickerUser32.NewProc("GetClassNameW").Call(hwnd, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
		if children[pid] && visible != 0 && windows.UTF16ToString(class) == "#32770" {
			found = hwnd
			return 0
		}
		return 1
	})
	_, _, _ = pickerUser32.NewProc("EnumWindows").Call(callback, 0)
	return found
}

func pickerChildProcesses() map[uint32]bool {
	result := map[uint32]bool{}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return result
	}
	defer func() { _ = windows.CloseHandle(snapshot) }()
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		if entry.ParentProcessID == uint32(os.Getpid()) {
			result[entry.ProcessID] = true
		}
	}
	return result
}
