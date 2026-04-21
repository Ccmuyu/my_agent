package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	LLM        LLMConfig        `mapstructure:"llm"`
	Perception PerceptionConfig `mapstructure:"perception"`
	Execution  ExecutionConfig  `mapstructure:"execution"`
	Browser    BrowserConfig    `mapstructure:"browser"`
	Server     ServerConfig     `mapstructure:"server"`
}

type LLMConfig struct {
	Provider    string  `mapstructure:"provider"`
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	BaseURL     string  `mapstructure:"base_url"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
}

type PerceptionConfig struct {
	ScreenshotDir string `mapstructure:"screenshot_dir"`
	OCREnabled    bool   `mapstructure:"ocr_enabled"`
}

type ExecutionConfig struct {
	MaxRetries        int `mapstructure:"max_retries"`
	RetryDelayMs      int `mapstructure:"retry_delay_ms"`
	ConfirmDangerous  bool `mapstructure:"confirm_dangerous"`
	ConfirmThreshold  int `mapstructure:"confirm_threshold"`
}

type BrowserConfig struct {
	Headless    bool   `mapstructure:"headless"`
	TimeoutMs   int    `mapstructure:"timeout_ms"`
	ViewportW   int    `mapstructure:"viewport_width"`
	ViewportH   int    `mapstructure:"viewport_height"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetEnvPrefix("AGENT")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 处理环境变量
	if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
		cfg.LLM.APIKey = apiKey
	}

	return &cfg, nil
}
