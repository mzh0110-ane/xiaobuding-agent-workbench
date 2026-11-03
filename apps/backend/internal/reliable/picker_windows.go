//go:build windows

package reliable

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func pickProject(ctx context.Context) (string, error) {
	script := `[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false)
$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Windows.Forms
$owner=New-Object System.Windows.Forms.Form
$dialog=New-Object System.Windows.Forms.FolderBrowserDialog
try {
  $owner.ShowInTaskbar=$false
  $owner.TopMost=$true
  $owner.Opacity=0
  $owner.StartPosition='CenterScreen'
  $owner.Size=New-Object System.Drawing.Size(1,1)
  $owner.Show()
  $owner.Activate()
  $dialog.Description='选择你要协作的项目文件夹'
  $dialog.ShowNewFolderButton=$false
  if($dialog.ShowDialog($owner) -eq [System.Windows.Forms.DialogResult]::OK){[Console]::Write($dialog.SelectedPath)}
} finally {
  $dialog.Dispose()
  $owner.Dispose()
}`
	powershell := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	cmd := exec.CommandContext(ctx, powershell, "-NoProfile", "-STA", "-Command", script)
	// Hide the console without hiding the interactive dialog owned by the form.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	raw, err := cmd.Output()
	return strings.TrimSpace(string(raw)), err
}
