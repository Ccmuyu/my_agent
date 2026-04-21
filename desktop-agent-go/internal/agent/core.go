package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"desktop-agent/internal/config"
	"desktop-agent/internal/llm"
	"desktop-agent/internal/tools"
)

const systemPrompt = `你是一个任务规划器。用户会给出一个任务，你需要将其分解为可执行的原子动作序列。

可用的工具：
- file_read: 读取文件内容，参数: path
- file_write: 写入文件内容，参数: path, content
- file_list: 列出目录文件，参数: path
- file_rename: 重命名/移动文件，参数: old, new
- file_delete: 删除文件，参数: path
- file_create_dir: 创建目录，参数: path
- browser_open: 打开URL，参数: url
- browser_click: 点击页面元素，参数: selector
- browser_input: 输入内容，参数: selector, text
- browser_screenshot: 网页截图，参数: path
- browser_close: 关闭浏览器
- shell_run: 执行Shell命令，参数: command
- screen_capture: 屏幕截图
- ocr_extract: OCR文字识别，参数: path

请将任务分解为JSON数组格式的动作序列。每个动作需要包含：
- tool: 工具名
- params: 参数字典
- risk_score: 风险评分(0-10)

只输出JSON数组，不要其他内容。`

type DesktopAgent struct {
	llm     llm.Client
	registry *tools.ToolRegistry
	config  *config.Config
	tasks   map[string]*Task
}

func NewDesktopAgent(llmClient llm.Client, registry *tools.ToolRegistry, cfg *config.Config) *DesktopAgent {
	return &DesktopAgent{
		llm:     llmClient,
		registry: registry,
		config:  cfg,
		tasks:   make(map[string]*Task),
	}
}

func (a *DesktopAgent) CreateTask(intent string, confirm bool) *Task {
	task := NewTask(intent)
	a.tasks[task.ID] = task
	go a.executeTask(task.ID, confirm)
	return task
}

func (a *DesktopAgent) GetTask(id string) (*Task, bool) {
	task, ok := a.tasks[id]
	return task, ok
}

func (a *DesktopAgent) ListTasks() []*Task {
	tasks := make([]*Task, 0, len(a.tasks))
	for _, t := range a.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

func (a *DesktopAgent) GetTools() []tools.ToolMeta {
	return a.registry.ListTools()
}

func (a *DesktopAgent) ConfirmTask(id string) (*Task, error) {
	task, ok := a.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found")
	}

	if task.Status != TaskStatusConfirming {
		return nil, fmt.Errorf("task not waiting for confirmation")
	}

	go a.executeTask(id, true)
	return task, nil
}

func (a *DesktopAgent) executeTask(taskID string, confirmed bool) {
	task := a.tasks[taskID]
	if task == nil {
		return
	}

	task.Status = TaskStatusRunning
	task.UpdatedAt = time.Now()

	// 调用LLM规划任务
	response, err := a.llm.Chat(task.Intent, systemPrompt)
	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = fmt.Sprintf("LLM调用失败: %v", err)
		task.UpdatedAt = time.Now()
		return
	}

	// 解析JSON动作
	actions, err := parseActions(response)
	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = fmt.Sprintf("解析动作失败: %v", err)
		task.UpdatedAt = time.Now()
		return
	}

	task.Actions = actions

	// 检查是否需要确认
	needConfirm := false
	for _, action := range actions {
		if action.RiskScore >= a.config.Execution.ConfirmThreshold {
			needConfirm = true
			break
		}
	}

	if needConfirm && !confirmed && !a.config.Execution.ConfirmDangerous {
		task.Status = TaskStatusConfirming
		task.UpdatedAt = time.Now()
		return
	}

	task.Confirmed = true

	// 执行动作
	for i, action := range actions {
		task.CurrentAction = i

		result := a.executeAction(action)
		task.Result = append(task.Result, result)

		if !result.Success {
			task.Status = TaskStatusFailed
			task.Error = result.Error
			task.UpdatedAt = time.Now()
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	task.Status = TaskStatusCompleted
	task.UpdatedAt = time.Now()
}

func (a *DesktopAgent) executeAction(action Action) ActionResult {
	maxRetries := a.config.Execution.MaxRetries

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := a.registry.Call(action.Tool, action.Params)
		if err != nil {
			if attempt < maxRetries {
				delay := time.Duration(a.config.Execution.RetryDelayMs) * time.Millisecond
				time.Sleep(delay)
				continue
			}
			return ActionResult{
				Success:    false,
				Error:      err.Error(),
				ActionIndex: action.Retry,
			}
		}

		return ActionResult{
			Success:    true,
			Output:    result,
			ActionIndex: action.Retry,
		}
	}

	return ActionResult{
		Success:    false,
		Error:      "max retries reached",
		ActionIndex: action.Retry,
	}
}

func parseActions(response string) ([]Action, error) {
	// 提取JSON数组
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("invalid JSON array format")
	}

	jsonStr := response[start : end+1]
	var actions []Action
	if err := json.Unmarshal([]byte(jsonStr), &actions); err != nil {
		return nil, fmt.Errorf("parse JSON failed: %w", err)
	}

	return actions, nil
}
