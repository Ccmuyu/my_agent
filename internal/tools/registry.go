package tools

import (
	"fmt"
	"time"
)

type ToolRegistry struct {
	tools   map[string]ToolFunc
	meta    map[string]ToolMeta
	history []ToolCall
}

type ToolFunc func(params map[string]any) (any, error)

type ToolMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RiskScore   int    `json:"risk_score"`
}

type ToolCall struct {
	Tool      string         `json:"tool"`
	Params    map[string]any `json:"params"`
	Timestamp time.Time      `json:"timestamp"`
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:   make(map[string]ToolFunc),
		meta:    make(map[string]ToolMeta),
		history: []ToolCall{},
	}
}

func (r *ToolRegistry) Register(name string, fn ToolFunc, desc string, riskScore int) {
	r.tools[name] = fn
	r.meta[name] = ToolMeta{
		Name:        name,
		Description: desc,
		RiskScore:   riskScore,
	}
}

func (r *ToolRegistry) Get(name string) (ToolFunc, bool) {
	fn, ok := r.tools[name]
	return fn, ok
}

func (r *ToolRegistry) ListTools() []ToolMeta {
	result := make([]ToolMeta, 0, len(r.meta))
	for _, m := range r.meta {
		result = append(result, m)
	}
	return result
}

func (r *ToolRegistry) Call(name string, params map[string]any) (any, error) {
	fn, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	result, err := fn(params)
	r.history = append(r.history, ToolCall{
		Tool:      name,
		Params:    params,
		Timestamp: time.Now(),
	})
	return result, err
}

func (r *ToolRegistry) GetHistory() []ToolCall {
	return r.history
}

func CreateRegistry() *ToolRegistry {
	r := NewToolRegistry()

	r.Register("file_read", fileRead, "读取文件内容", 1)
	r.Register("file_write", fileWrite, "写入文件内容", 2)
	r.Register("file_list", fileList, "列出目录文件", 1)
	r.Register("file_rename", fileRename, "重命名/移动文件", 3)
	r.Register("file_delete", fileDelete, "删除文件", 8)
	r.Register("file_create_dir", fileCreateDir, "创建目录", 2)
	r.Register("file_glob", fileGlob, "搜索文件", 1)
	r.Register("file_grep", fileGrep, "搜索文件内容", 1)
	r.Register("shell_run", shellRun, "执行Shell命令", 5)
	r.Register("screen_capture", screenCapture, "屏幕截图", 1)
	r.Register("ocr_extract", ocrExtract, "OCR文字识别", 1)

	r.Register("browser_open", browserOpen, "打开URL", 1)
	r.Register("browser_click", browserClick, "点击页面元素", 2)
	r.Register("browser_input", browserInput, "输入内容", 2)
	r.Register("browser_scroll", browserScroll, "滚动页面", 2)
	r.Register("browser_screenshot", browserScreenshot, "网页截图", 1)
	r.Register("browser_close", browserClose, "关闭浏览器", 1)

	r.Register("weather", weatherQuery, "查询天气", 1)

	return r
}
