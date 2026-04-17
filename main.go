package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
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
	SKILLS_DIR = "./skills"
)

// ================= Provider 配置 =================
type ProviderConfig struct {
	Name    string
	APIKey  string
	BaseURL string
	Model   string
}

var (
	providers = map[string]ProviderConfig{
		"zhipu": {
			Name:    "智谱AI",
			APIKey:  "",
			BaseURL: "https://open.bigmodel.cn/api/paas/v4/",
			Model:   "glm-4-flash",
		},
		"qwen": {
			Name:    "阿里云百炼",
			APIKey:  "",
			BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
			Model:   "qwen-turbo",
		},
		"doubao": {
			Name:    "火山引擎豆包",
			APIKey:  "",
			BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
			Model:   "Doubao-Seed-2.0-lite",
		},
		"spark": {
			Name:    "讯飞星火",
			APIKey:  "",
			BaseURL: "https://spark-api.xf-yun.com/v3.1/chat",
			Model:   "spark-lite",
		},
	}

	currentProvider string
	apiKey          string
)

func init() {
	flag.StringVar(&currentProvider, "provider", "zhipu", "LLM provider: zhipu, qwen, doubao, spark")
	flag.StringVar(&apiKey, "api-key", "", "API Key for the selected provider")
}

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

// doWeather 内部实现的天气查询逻辑
func doWeather(args map[string]interface{}) (string, error) {
	city, _ := args["city"].(string)

	if city == "" {
		ipServices := []string{
			"https://ipapi.co/json/",
			"https://ipinfo.io/json",
		}

		for _, ipURL := range ipServices {
			ipResp, err := http.Get(ipURL)
			if err != nil {
				continue
			}

			var ipResult map[string]interface{}
			if err := json.NewDecoder(ipResp.Body).Decode(&ipResult); err != nil {
				ipResp.Body.Close()
				continue
			}
			ipResp.Body.Close()

			city, _ = ipResult["city"].(string)
			if city != "" {
				break
			}
		}

		if city == "" {
			city = "Beijing"
		}
	}

	var weatherResult map[string]interface{}
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		apiURL := fmt.Sprintf("https://wttr.in/%s?format=j1", url.QueryEscape(city))
		resp, err := http.Get(apiURL)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if err := json.NewDecoder(resp.Body).Decode(&weatherResult); err != nil {
			resp.Body.Close()
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		resp.Body.Close()

		if weatherResult != nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if weatherResult == nil {
		if lastErr != nil {
			return fmt.Sprintf("❌ 天气查询失败: %v\n🌍 建议: 请提供具体城市名称，例如 'weather(city: \"北京\")'", lastErr), nil
		}
		return "❌ 无法获取天气数据\n🌍 建议: 请提供具体城市名称，例如 'weather(city: \"北京\")'", nil
	}

	current, ok := weatherResult["current_condition"].([]interface{})
	if !ok || len(current) == 0 {
		return "❌ 天气数据格式错误\n🌍 建议: 请提供具体城市名称，例如 'weather(city: \"北京\")'", nil
	}

	c, ok := current[0].(map[string]interface{})
	if !ok {
		return "❌ 天气数据解析失败\n🌍 建议: 请提供具体城市名称，例如 'weather(city: \"北京\")'", nil
	}

	temp, _ := c["temp_C"].(string)
	feelsLike, _ := c["FeelsLikeC"].(string)
	humidity, _ := c["humidity"].(string)
	windSpeed, _ := c["windspeedKmph"].(string)
	windDir, _ := c["winddir16Point"].(string)

	weatherDesc := "未知"
	if weatherDescs, ok := c["weatherDesc"].([]interface{}); ok && len(weatherDescs) > 0 {
		if desc, ok := weatherDescs[0].(map[string]interface{}); ok {
			if value, ok := desc["value"].(string); ok {
				weatherDesc = value
			}
		}
	}

	forecast, hasForecast := weatherResult["weather"].([]interface{})
	tomorrow := ""
	if hasForecast && len(forecast) > 1 {
		if day2, ok := forecast[1].(map[string]interface{}); ok {
			if dayDesc, ok := day2["weatherDesc"].([]interface{}); ok && len(dayDesc) > 0 {
				if desc, ok := dayDesc[0].(map[string]interface{}); ok {
					if value, ok := desc["value"].(string); ok {
						tomorrow = value
					}
				}
			}
			if maxTemp, ok := day2["maxtempC"].(string); ok {
				if minTemp, ok := day2["mintempC"].(string); ok {
					tomorrow = fmt.Sprintf("明天: %s，温度 %s~%s°C", tomorrow, minTemp, maxTemp)
				}
			}
		}
	}

	result := fmt.Sprintf("📍 %s 天气\n🌡️ 温度: %s°C (体感 %s°C)\n🌤️ 天气: %s\n💧 湿度: %s%%\n💨 风速: %s km/h %s",
		city, temp, feelsLike, weatherDesc, humidity, windSpeed, windDir)

	if tomorrow != "" {
		result += "\n" + tomorrow
	}

	return result, nil
}

func ExecuteSkill(name string, args map[string]interface{}) (string, error) {
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

	if name == "translate_text" {
		return doTranslate(args)
	}

	if name == "weather" {
		return doWeather(args)
	}

	skillMutex.RLock()
	_, exists := skillRegistry[name]
	skillMutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("skill not found: %s", name)
	}

	cmdStr, ok := args["command"].(string)
	if !ok || cmdStr == "" {
		return "", fmt.Errorf("missing 'command' argument for skill '%s'", name)
	}

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
	flag.Parse()

	provider, ok := providers[currentProvider]
	if !ok {
		fmt.Printf("❌ Unknown provider: %s\n", currentProvider)
		fmt.Println("Available providers: zhipu, qwen, doubao, spark")
		os.Exit(1)
	}

	if apiKey == "" {
		fmt.Printf("❌ API key is required for provider '%s'\n", currentProvider)
		fmt.Printf("Usage: go run main.go -provider=%s -api-key=YOUR_API_KEY\n", currentProvider)
		os.Exit(1)
	}

	provider.APIKey = apiKey

	fmt.Printf("🔍 Initializing Agent with %s (%s)...\n", provider.Name, provider.Model)

	if err := LoadSkillsFromDir(SKILLS_DIR); err != nil {
		log.Fatalf("Failed to load initial skills: %v", err)
	}

	config := openai.DefaultConfig(provider.APIKey)
	config.BaseURL = provider.BaseURL
	client := openai.NewClientWithConfig(config)

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

	fmt.Printf("\n🚀 Dynamic Skill Agent Started (%s)\n", provider.Name)
	fmt.Println("Type 'quit' to exit.")

	for {
		fmt.Print("👤 User: ")
		var input string
		fmt.Scanln(&input)
		if strings.ToLower(input) == "quit" {
			break
		}
		if input == "" {
			continue
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleUser, Content: input,
		})

		fmt.Println("🔄 Thinking...")

		for step := 0; step < 5; step++ {
			modelName := provider.Model
			if currentProvider == "qwen" {
				modelName = "qwen-plus"
			}

			resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
				Model:    modelName,
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
