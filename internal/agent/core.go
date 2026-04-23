package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Ccmuyu/my_agent/internal/config"
	"github.com/Ccmuyu/my_agent/internal/llm"
	"github.com/Ccmuyu/my_agent/internal/tools"
)

const systemPrompt = `你是一个智能助手。

用户提问时，使用技能处理。

技能格式（严格JSON）：
[{"skill": "weather", "args": {"city": "北京"}}]
[{"skill": "translate_text", "args": {"text": "hello", "target_lang": "zh"}}]

示例：
问：北京天气怎么样
答：[{"skill": "weather", "args": {"city": "北京"}}]

问：翻译hello到中文
答：[{"skill": "translate_text", "args": {"text": "hello", "target_lang": "zh"}}]

只输出JSON数组，禁止任何其他文字。`

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

func (a *DesktopAgent) GetToolHistory() []tools.ToolCall {
	return a.registry.GetHistory()
}

func (a *DesktopAgent) StreamTaskUpdate(taskID string, content string) {
	task, ok := a.tasks[taskID]
	if !ok {
		return
	}
	task.Thinking += content
	task.UpdatedAt = time.Now()
}

func (a *DesktopAgent) StreamCreateTask(intent string, confirm bool, onChunk func(string)) *Task {
	task := NewTask(intent)
	a.tasks[task.ID] = task
	go a.streamExecuteTask(task.ID, confirm, onChunk)
	return task
}

func (a *DesktopAgent) streamExecuteTask(taskID string, confirmed bool, onChunk func(string)) {
	task := a.tasks[taskID]
	if task == nil {
		return
	}

	task.Status = TaskStatusRunning
	task.UpdatedAt = time.Now()

	onChunk("🔄 正在分析任务...\n")

	response, err := a.llm.Chat(task.Intent, systemPrompt)
	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = fmt.Sprintf("LLM调用失败: %v", err)
		task.UpdatedAt = time.Now()
		onChunk(fmt.Sprintf("❌ 错误: %s\n", err))
		return
	}

	onChunk("✅ 任务分析完成\n")

	actions, textResponse, err := parseActions(response)
	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = fmt.Sprintf("解析动作失败: %v", err)
		task.UpdatedAt = time.Now()
		onChunk(fmt.Sprintf("❌ 解析失败: %s\n", err))
		return
	}

	// 如果没有动作（模型直接回答问题），直接返回结果
	if actions == nil || len(actions) == 0 {
		task.Status = TaskStatusCompleted
		task.UpdatedAt = time.Now()
		output := textResponse
		if output == "" {
			output = response
		}
		onChunk(fmt.Sprintf("💬 %s\n", output))
		onChunk("\n✨ 任务完成!")
		return
	}

	task.Actions = actions
	onChunk(fmt.Sprintf("📋 计划执行 %d 个动作\n\n", len(actions)))

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
		onChunk("⚠️ 需要确认执行高风险操作\n")
		return
	}

	task.Confirmed = true

	for i, action := range actions {
		task.CurrentAction = i

		// 调用工具或技能
		var result ActionResult
		if action.Skill != "" {
			onChunk(fmt.Sprintf("🛠️ Calling: [%s]\n", action.Skill))
			result = a.executeSkill(action)
		} else if action.Tool != "" {
			onChunk(fmt.Sprintf("🛠️ 执行: %s\n", action.Tool))
			result = a.executeAction(action)
		}
		task.Result = append(task.Result, result)

		if result.Success {
			onChunk(fmt.Sprintf("  ✅ 成功\n"))
		} else {
			task.Status = TaskStatusFailed
			task.Error = result.Error
			task.UpdatedAt = time.Now()
			onChunk(fmt.Sprintf("  ❌ 失败: %s\n", result.Error))
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	task.Status = TaskStatusCompleted
	task.UpdatedAt = time.Now()
	onChunk("\n✨ 任务完成!")
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

func (a *DesktopAgent) CancelTask(id string) error {
	task, ok := a.tasks[id]
	if !ok {
		return fmt.Errorf("task not found")
	}

	if task.Status == TaskStatusCompleted || task.Status == TaskStatusFailed || task.Status == TaskStatusCancelled {
		return fmt.Errorf("task already finished")
	}

	task.Status = TaskStatusCancelled
	task.UpdatedAt = time.Now()
	return nil
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
	actions, _, err := parseActions(response)
	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = fmt.Sprintf("解析动作失败: %v", err)
		task.UpdatedAt = time.Now()
		return
	}

	// 如果没有动作（直接回答问题）
	if actions == nil || len(actions) == 0 {
		task.Status = TaskStatusCompleted
		task.UpdatedAt = time.Now()
		task.Result = []ActionResult{{
			Success: true,
			Output:  response,
		}}
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

func (a *DesktopAgent) executeSkill(action Action) ActionResult {
	skillName := action.Skill
	args := action.Args

	switch skillName {
	case "weather":
		result, err := tools.WeatherQuery(args)
		if err != nil {
			return ActionResult{Success: false, Error: err.Error()}
		}
		return ActionResult{Success: true, Output: result}
	default:
		return ActionResult{Success: false, Error: "unknown skill: " + skillName}
	}
}

func parseActions(response string) ([]Action, string, error) {
	response = strings.TrimSpace(response)

	response = strings.ReplaceAll(response, "'", "\"")

	var actions []Action

	re := regexp.MustCompile(`\{[^{}]*\}`)
	matches := re.FindAllString(response, -1)

	for _, match := range matches {
		match = strings.TrimSpace(match)
		if match == "" || match == "{}" {
			continue
		}
		var action Action
		if err := json.Unmarshal([]byte(match), &action); err == nil && (action.Skill != "" || action.Tool != "") {
			if action.Skill == "weather" && action.Args != nil {
				if city, ok := action.Args["city"].(string); ok {
					if strings.Contains(city, "location") || city == "" {
						action.Args["city"] = ""
					}
				}
			}
			actions = append(actions, action)
		}
	}

	if len(actions) == 0 {
		return nil, response, nil
	}

	return actions, "", nil
}
