package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Ccmuyu/my_agent/internal/agent"
	"github.com/Ccmuyu/my_agent/internal/config"
	"github.com/Ccmuyu/my_agent/internal/llm"
	"github.com/Ccmuyu/my_agent/internal/rag"
	"github.com/Ccmuyu/my_agent/internal/tools"
	"github.com/Ccmuyu/my_agent/internal/api"
)

var (
	configPath = flag.String("config", "config.yaml", "config file path")
	mode       = flag.String("mode", "cli", "run mode: cli or web")
	port       = flag.Int("port", 8000, "web server port")
	apiKey     = flag.String("api-key", "", "LLM API key (优先级: CLI > env > config)")
	model      = flag.String("model", "", "LLM model")
	provider   = flag.String("provider", "", "LLM provider")
	baseURL    = flag.String("base-url", "", "LLM base URL")
)

func main() {
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("config load failed, using defaults: %v", err)
		cfg = &config.Config{
			Server: config.ServerConfig{Host: "0.0.0.0", Port: 8000},
			LLM: config.LLMConfig{
				Provider: "openrouter",
				Model: "glm-4-flash",
				BaseURL: "https://open.bigmodel.cn/api/paas/v4",
			},
			Execution: config.ExecutionConfig{
				MaxRetries: 3,
				RetryDelayMs: 1000,
				ConfirmDangerous: true,
				ConfirmThreshold: 5,
			},
		}
	}

	resolveLLMConfig(cfg)

	llmClient := llm.NewOpenRouterClient(
		cfg.LLM.APIKey,
		cfg.LLM.Model,
		cfg.LLM.BaseURL,
		cfg.LLM.Temperature,
		cfg.LLM.MaxTokens,
	)

	registry := tools.CreateRegistry()

	if cfg.RAG.Enabled {
		ctx := context.Background()
		ragService, err := rag.NewRAGServiceFromConfig(ctx, &cfg.RAG)
		if err != nil {
			log.Printf("RAG service init failed: %v", err)
		} else {
			ragTool := tools.NewRAGTool(ragService, ctx)
			ragTool.RegisterToRegistry(registry)
		}
	}

	desktopAgent := agent.NewDesktopAgent(llmClient, registry, cfg)

	if *mode == "cli" {
		runCLI(desktopAgent)
	} else {
		api.Run(desktopAgent, cfg.Server.Host, cfg.Server.Port)
	}
}

func runCLI(a *agent.DesktopAgent) {
	fmt.Println("Desktop Agent CLI")
	fmt.Println("输入任务描述，或 'quit' 退出")
	fmt.Printf("可用工具: ")
	tools := a.GetTools()
	for i, t := range tools {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(t.Name)
	}
	fmt.Println()
	fmt.Println()

	var input string
	for {
		fmt.Print("> ")
		if _, err := fmt.Scanln(&input); err != nil {
			continue
		}

		if input == "quit" || input == "exit" || input == "q" {
			break
		}

		if input == "" {
			continue
		}

		task := a.CreateTask(input, false)
		fmt.Printf("任务创建成功: %s\n", task.ID)
		fmt.Printf("状态: %s\n", task.Status)

		for {
			t, _ := a.GetTask(task.ID)
			if t.Status == agent.TaskStatusCompleted ||
				t.Status == agent.TaskStatusFailed ||
				t.Status == agent.TaskStatusConfirming {
				break
			}
		}

		t, _ := a.GetTask(task.ID)
		fmt.Printf("最终状态: %s\n", t.Status)

		if t.Error != "" {
			fmt.Printf("错误: %s\n", t.Error)
		}

		if len(t.Result) > 0 {
			fmt.Println("执行结果:")
			for _, r := range t.Result {
				if r.Success {
					fmt.Printf("  ✓ %v\n", r.Output)
				} else {
					fmt.Printf("  ✗ %s\n", r.Error)
				}
			}
		}
		fmt.Println()
	}

	fmt.Println("Bye!")
	os.Exit(0)
}

func resolveLLMConfig(cfg *config.Config) {
	resolved := cfg.LLM

	if *apiKey != "" {
		resolved.APIKey = *apiKey
	} else if k := os.Getenv("LLM_API_KEY"); k != "" {
		resolved.APIKey = k
	}

	if *model != "" {
		resolved.Model = *model
	} else if m := os.Getenv("LLM_MODEL"); m != "" {
		resolved.Model = m
	}

	if *baseURL != "" {
		resolved.BaseURL = *baseURL
	} else if u := os.Getenv("LLM_BASE_URL"); u != "" {
		resolved.BaseURL = u
	}

	if *provider != "" {
		resolved.Provider = *provider
	}

	cfg.LLM = resolved
	log.Printf("LLM: %s - %s (%s)", cfg.LLM.Provider, cfg.LLM.Model, cfg.LLM.BaseURL)
}