package tools

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func shellRun(params map[string]any) (any, error) {
	command, ok := params["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing command parameter")
	}

	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Sprintf("error: %s", string(output)), err
	}

	return string(output), nil
}

func screenCapture(params map[string]any) (any, error) {
	dir := "/tmp/desktop-agent/screenshots"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create dir failed: %w", err)
	}

	filename := fmt.Sprintf("screen_%s.png", time.Now().Format("20060102_150405"))
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

func ocrExtract(params map[string]any) (any, error) {
	imagePath, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing path parameter")
	}

	cmd := exec.Command("tesseract", imagePath, "-")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tesseract failed: %w", err)
	}

	return string(output), nil
}