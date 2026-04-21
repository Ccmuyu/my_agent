package tools

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

var browserPID = 0

func browserOpen(params map[string]any) (any, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing url parameter")
	}

	cmd := exec.Command("xdg-open", url)
	if err := cmd.Start(); err != nil {
		cmd = exec.Command("google-chrome", url)
		if err := cmd.Start(); err != nil {
			cmd = exec.Command("firefox", url)
			if err := cmd.Start(); err != nil {
				return nil, fmt.Errorf("no browser found: %w", err)
			}
		}
	}

	return fmt.Sprintf("已打开: %s", url), nil
}

func browserScreenshot(params map[string]any) (any, error) {
	dir := "/tmp/desktop-agent/screenshots"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create dir failed: %w", err)
	}

	filename := fmt.Sprintf("browser_%d.png", time.Now().Unix())
	path := fmt.Sprintf("%s/%s", dir, filename)

	cmd := exec.Command("gnome-screenshot", "-f", path)
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("scrot", path)
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("screenshot failed: %w", err)
		}
	}

	return path, nil
}

func browserClose(params map[string]any) (any, error) {
	if browserPID > 0 {
		cmd := exec.Command("kill", fmt.Sprintf("%d", browserPID))
		cmd.Run()
		browserPID = 0
	}
	return "浏览器已关闭", nil
}
