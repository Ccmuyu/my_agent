package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"desktop-agent/internal/agent"
	"desktop-agent/internal/config"
	"desktop-agent/internal/llm"
	"desktop-agent/internal/tools"
	"desktop-agent/internal/api"
)

var (
	configPath = flag.String("config", "config.yaml", "config file path")
	mode       = flag.String("mode", "cli", "run mode: cli or web")
	port       = flag.Int("port", 8000, "web server port")
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
				Model: "anthropic/claude-3.5-sonnet",
				BaseURL: "https://openrouter.ai/api/v1",
			},
			Execution: config.ExecutionConfig{
				MaxRetries: 3,
				RetryDelayMs: 1000,
				ConfirmDangerous: true,
				ConfirmThreshold: 5,
			},
		}
	}

	// 初始化LLM客户端
	var llmClient llm.Client
	switch cfg.LLM.Provider {
	case "openrouter":
		llmClient = llm.NewOpenRouterClient(
			cfg.LLM.APIKey,
			cfg.LLM.Model,
			cfg.LLM.BaseURL,
			cfg.LLM.Temperature,
			cfg.LLM.MaxTokens,
		)
	default:
		log.Fatalf("unsupported LLM provider: %s", cfg.LLM.Provider)
	}

	// 创建工具注册表
	registry := tools.CreateRegistry()

	// 创建Agent
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

		// 等待执行完成
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