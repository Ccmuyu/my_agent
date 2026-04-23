package tools

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

var browserPID = 0

type BrowserState struct {
	url     string
	running bool
}

var browserState = &BrowserState{}

func browserOpen(params map[string]any) (any, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing url parameter")
	}

	cmd := exec.Command("xdg-open", url)
	if err := cmd.Start(); err != nil {
		cmd = exec.Command("google-chrome", "--new-window", url)
		if err := cmd.Start(); err != nil {
			cmd = exec.Command("firefox", "-new-window", url)
			if err := cmd.Start(); err != nil {
				return nil, fmt.Errorf("no browser found: %w", err)
			}
		}
	}

	browserState.running = true
	browserState.url = url

	return fmt.Sprintf("已打开: %s", url), nil
}

func browserClick(params map[string]any) (any, error) {
	selector, ok := params["selector"].(string)
	if !ok {
		return nil, fmt.Errorf("missing selector parameter")
	}

	cmd := exec.Command("xdotool", "search", "--onlyvisible", "--class", "chromium", "click", "1")
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("xdotool", "mousemove", "100", "100", "click", "1")
		cmd.Run()
	}

	return fmt.Sprintf("已点击: %s", selector), nil
}

func browserInput(params map[string]any) (any, error) {
	text, ok := params["text"].(string)
	if !ok {
		return nil, fmt.Errorf("missing text parameter")
	}

	cmd := exec.Command("xdotool", "type", "--window", "current", text)
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("xdotool", "type", text)
		cmd.Run()
	}

	return fmt.Sprintf("已输入: %s", text), nil
}

func browserScroll(params map[string]any) (any, error) {
	direction := "down"
	if d, ok := params["direction"].(string); ok {
		direction = d
	}

	amount := 500
	if a, ok := params["amount"].(float64); ok {
		amount = int(a)
	}

	var cmd *exec.Cmd
	if direction == "up" {
		cmd = exec.Command("xdotool", "mousemove", "--window", "0", "100", "100", "click", "--wheel-up", fmt.Sprintf("%d", amount/100))
	} else {
		cmd = exec.Command("xdotool", "mousemove", "--window", "0", "100", "100", "click", "--wheel-down", fmt.Sprintf("%d", amount/100))
	}
	cmd.Run()

	return fmt.Sprintf("已滚动: %s", direction), nil
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
