package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

func fileRead(params map[string]any) (any, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing path parameter")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	return string(data), nil
}

func fileWrite(params map[string]any) (any, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing path parameter")
	}

	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("missing content parameter")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create dir failed: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("write file failed: %w", err)
	}

	return fmt.Sprintf("已写入: %s", path), nil
}

func fileList(params map[string]any) (any, error) {
	path := "."
	if p, ok := params["path"].(string); ok {
		path = p
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir failed: %w", err)
	}

	result := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		result = append(result, map[string]any{
			"name":     e.Name(),
			"type":    "dir",
			"size":    info.Size(),
			"modified": info.ModTime().Unix(),
		})
	}

	return result, nil
}

func fileRename(params map[string]any) (any, error) {
	oldPath, ok := params["old"].(string)
	if !ok {
		return nil, fmt.Errorf("missing old parameter")
	}

	newPath, ok := params["new"].(string)
	if !ok {
		return nil, fmt.Errorf("missing new parameter")
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return nil, fmt.Errorf("rename failed: %w", err)
	}

	return fmt.Sprintf("已重命名: %s -> %s", oldPath, newPath), nil
}

func fileDelete(params map[string]any) (any, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing path parameter")
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	if info.IsDir() {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}

	if err != nil {
		return nil, fmt.Errorf("delete failed: %w", err)
	}

	return fmt.Sprintf("已删除: %s", path), nil
}

func fileCreateDir(params map[string]any) (any, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing path parameter")
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("create dir failed: %w", err)
	}

	return fmt.Sprintf("已创建目录: %s", path), nil
}