package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v3"
)

// ================= 配置区域 =================
const (
	ZHIPU_API_KEY = "34a2e4fbcd0d4ad3af5a0a9b54dc3e1f.HkRxvndaEepC8ga7" // ⚠️请替换
	BASE_URL      = "https://open.bigmodel.cn/api/paas/v4/"
	MODEL_NAME    = "glm-4-flash"
	SKILLS_DIR    = "./skills"
)

// ================= 数据结构 =================

type SkillMeta struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Parameters  map[string]interface{} `yaml:"parameters"`
}

type SkillFile struct {
	Meta   SkillMeta
	Prompt string
}

var (
	skillRegistry = make(map[string]*SkillFile)
	skillMutex    sync.RWMutex
)

// ================= 核心逻辑 =================

func LoadSkillsFromDir(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return err
	}

	newRegistry := make(map[string]*SkillFile)
	loadedCount := 0

	for _, file := range files {
		skill, err := parseSkillFile(file)
		if err != nil {
			log.Printf("⚠️ Skip %s: %v\n", file, err)
			continue
		}
		newRegistry[skill.Meta.Name] = skill
		loadedCount++
	}

	skillMutex.Lock()
	skillRegistry = newRegistry
	skillMutex.Unlock()

	fmt.Printf("✅ Reloaded %d skills from %s\n", loadedCount, dir)
	return nil
}

func parseSkillFile(path string) (*SkillFile, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	text := string(content)
	if !strings.HasPrefix(text, "---") {
		return nil, fmt.Errorf("missing frontmatter")
	}

	endIndex := strings.Index(text[3:], "---")
	if endIndex == -1 {
		return nil, fmt.Errorf("unclosed frontmatter")
	}

	yamlPart := text[3 : 3+endIndex]
	promptPart := strings.TrimSpace(text[3+endIndex+3:])

	var meta SkillMeta
	if err := yaml.Unmarshal([]byte(yamlPart), &meta); err != nil {
		return nil, err
	}

	if meta.Name == "" {
		return nil, fmt.Errorf("missing name in metadata")
	}

	return &SkillFile{
		Meta:   meta,
		Prompt: promptPart,
	}, nil
}

func GetToolsForLLM() []openai.Tool {
	skillMutex.RLock()
	defer skillMutex.RUnlock()

	var tools []openai.Tool
	for _, skill := range skillRegistry {
		paramJSON, _ := json.Marshal(skill.Meta.Parameters)
		fullDesc := fmt.Sprintf("%s\n\nInstructions:\n%s", skill.Meta.Description, skill.Prompt)
		
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        skill.Meta.Name,
				Description: fullDesc,
				Parameters:  json.RawMessage(paramJSON),
			},
		})
	}
	return tools
}

// doTranslate 内部实现的翻译逻辑
func doTranslate(args map[string]interface{}) (string, error) {
	text, ok := args["text"].(string)
	if !ok {
		return "", fmt.Errorf("missing text argument")
	}
	
	targetLang, ok := args["target_lang"].(string)
	if !ok {
		return "", fmt.Errorf("missing target_lang argument")
	}

	sourceLang, _ := args["source_lang"].(string)
	if sourceLang == "" {
		sourceLang = "Auto"
	}

	// 使用 MyMemory Free API
	baseURL := "https://api.mymemory.translated.net/get"
	params := url.Values{}
	params.Add("q", text)
	params.Add("langpair", fmt.Sprintf("%s|%s", sourceLang, targetLang))

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	resp, err := http.Get(reqURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 解析 JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if responseData, ok := result["responseData"].(map[string]interface{}); ok {
		if translatedText, ok := responseData["translatedText"].(string); ok {
			return translatedText, nil
		}
	}

	return fmt.Sprintf("API Error: %s", string(body)), nil
}

func ExecuteSkill(name string, args map[string]interface{}) (string, error) {
	// 1. 内部特殊技能：重载
	if name == "reload_skills" {
		err := LoadSkillsFromDir(SKILLS_DIR)
		if err != nil {
			return fmt.Sprintf("Failed to reload skills: %v", err), nil
		}
		
		skillMutex.RLock()
		var skillList []string
		for k := range skillRegistry {
			skillList = append(skillList, k)
		}
		skillMutex.RUnlock()
		
		return fmt.Sprintf("✅ Skills reloaded successfully. Available skills: %s", strings.Join(skillList, ", ")), nil
	}

	// 2. 内部特殊技能：翻译
	if name == "translate_text" {
		return doTranslate(args)
	}

	// 3. 常规技能检查
	skillMutex.RLock()
	_, exists := skillRegistry[name]
	skillMutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("skill not found: %s", name)
	}

	// 4. 通用 Shell 执行逻辑
	cmdStr, ok := args["command"].(string)
	if !ok || cmdStr == "" {
		return "", fmt.Errorf("missing 'command' argument for skill '%s'", name)
	}

	// 5. 安全白名单
	allowedBaseCommands := map[string]bool{
		"git": true, "ls": true, "pwd": true, "cat": true,
		"head": true, "tail": true, "grep": true, "find": true,
		"df": true, "free": true, "ps": true, "date": true,
		"whoami": true, "uname": true, "echo": true,
		"curl": true, "jq": true,
	}

	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	baseCmd := parts[0]
	if !allowedBaseCommands[baseCmd] {
		return "", fmt.Errorf("🚫 Security Block: Command '%s' is not allowed.", baseCmd)
	}

	// 6. 执行
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, baseCmd, parts[1:]...)
	
	if path, ok := args["path"].(string); ok && path != "" {
		cmd.Dir = path
	}

	output, err := cmd.CombinedOutput()
	result := string(output)
	
	if err != nil {
		return fmt.Sprintf("⚠️ Execution Error: %v\nOutput:\n%s", err, result), nil
	}

	if len(result) > 4000 {
		result = result[:4000] + "\n... (truncated)"
	}

	return result, nil
}

// ================= 主程序 =================

func main() {
	fmt.Println("🔍 Initializing Agent...")
	
	if err := LoadSkillsFromDir(SKILLS_DIR); err != nil {
		log.Fatalf("Failed to load initial skills: %v", err)
	}

	config := openai.DefaultConfig(ZHIPU_API_KEY)
	config.BaseURL = BASE_URL
	client := openai.NewClientWithConfig(config)

	// ✅ 优化后的 System Prompt，强调不要拆分任务
	messages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: `You are a helpful assistant with dynamic skills. 
IMPORTANT RULES:
1. If the user provides a phrase or sentence (especially in quotes like 'Hello World'), treat it as a SINGLE unit. 
2. DO NOT split the text into multiple tool calls. Call 'translate_text' ONCE with the full text.
3. Use 'reload_skills' if new skills are added.`,
		},
	}

	fmt.Println("\n🚀 Dynamic Skill Agent Started")
	fmt.Println("Type 'quit' to exit.\n")

	for {
		fmt.Print("👤 User: ")
		var input string
		fmt.Scanln(&input)
		if strings.ToLower(input) == "quit" { break }
		if input == "" { continue }

		messages = append(messages, openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleUser, Content: input,
		})

		fmt.Println("🔄 Thinking...")

		for step := 0; step < 5; step++ {
			resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
				Model:    MODEL_NAME,
				Messages: messages,
				Tools:    GetToolsForLLM(),
			})
			if err != nil {
				log.Printf("❌ LLM Error: %v\n", err)
				break
			}

			msg := resp.Choices[0].Message
			if len(msg.ToolCalls) == 0 {
				fmt.Printf("✅ Assistant: %s\n\n", msg.Content)
				messages = append(messages, msg)
				break
			}

			fmt.Printf("🛠️  Calling: ")
			for _, tc := range msg.ToolCalls {
				fmt.Printf("[%s] ", tc.Function.Name)
			}
			fmt.Println()

			messages = append(messages, msg)

			for _, tc := range msg.ToolCalls {
				var funcArgs map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &funcArgs)
				
				result, err := ExecuteSkill(tc.Function.Name, funcArgs)
				if err != nil {
					result = fmt.Sprintf("System Error: %v", err)
				}

				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					Name:       tc.Function.Name,
					ToolCallID: tc.ID,
				})
			}
		}
	}
}

