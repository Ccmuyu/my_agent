# Dynamic Skill Agent

一个基于 Go 语言和智谱 AI 的动态技能 Agent，支持多种实用技能如天气查询、翻译、系统信息等。

## ✨ 功能特性

- **动态技能管理**：自动加载 `skills` 目录下的技能定义
- **天气查询**：支持自动 IP 定位和手动指定城市，提供当前和未来天气
- **翻译功能**：支持多语言互译
- **系统信息**：查看系统资源使用情况
- **Git 操作**：执行 Git 相关命令
- **技能重载**：无需重启即可加载新技能

## 🚀 快速开始

### 环境要求
- Go 1.18+
- 智谱 AI API Key（用于 LLM 调用）

### 安装

```bash
git clone https://github.com/yourusername/dynamic-skill-agent.git
cd dynamic-skill-agent
go mod tidy
```

### 配置

编辑 `main.go` 文件，设置你的智谱 AI API Key：

```go
const (
	ZHIPU_API_KEY = "your-api-key-here" // 替换为你的 API Key
	BASE_URL      = "https://open.bigmodel.cn/api/paas/v4/"
	MODEL_NAME    = "glm-4-flash"
	SKILLS_DIR    = "./skills"
)
```

### 运行

```bash
go run main.go
```

## 📚 技能列表

| 技能名称 | 描述 | 参数 | 示例 |
|---------|------|------|------|
| `weather` | 查询天气状态 | `city` (可选) | `weather(city: "北京")` |
| `translate_text` | 翻译文本 | `text`, `target_lang`, `source_lang` (可选) | `translate_text(text: "Hello", target_lang: "zh")` |
| `system_info` | 查看系统信息 | `command` | `system_info(command: "df -h")` |
| `git_operations` | 执行 Git 命令 | `command`, `path` (可选) | `git_operations(command: "status")` |
| `reload_skills` | 重载技能 | 无 | `reload_skills()` |

## 🎯 使用示例

### 1. 查询天气

```
👤 User: 北京今天天气怎么样？
🔄 Thinking...
🛠️  Calling: [weather]
✅ Assistant: 📍 北京 天气
🌡️ 温度: 18°C (体感 17°C)
🌤️ 天气: Partly cloudy
💧 湿度: 45%
💨 风速: 12 km/h NE
明天: Partly cloudy，温度 16~20°C
```

### 2. 翻译文本

```
👤 User: 翻译 "Hello world" 到中文
🔄 Thinking...
🛠️  Calling: [translate_text]
✅ Assistant: 你好，世界
```

### 3. 查看系统信息

```
👤 User: 查看磁盘使用情况
🔄 Thinking...
🛠️  Calling: [system_info]
✅ Assistant: Filesystem      Size  Used Avail Use% Mounted on
/dev/sda1        50G   20G   28G  42% /
```

### 4. 执行 Git 命令

```
👤 User: 查看 Git 状态
🔄 Thinking...
🛠️  Calling: [git_operations]
✅ Assistant: On branch main
Your branch is up to date with 'origin/main'.

Changes not staged for commit:
  (use "git add <file>..." to update what will be committed)
  (use "git restore <file>..." to discard changes in working directory)
	modified:   main.go

no changes added to commit (use "git add" and/or "git commit -a")
```

## 🔧 自定义技能

### 创建新技能

1. 在 `skills` 目录下创建新的 `.md` 文件，例如 `example.md`

2. 按照以下格式编写技能定义：

```markdown
---
name: "example"
description: "示例技能，用于演示如何创建自定义技能"
parameters:
  type: object
  properties:
    message:
      type: string
      description: "要处理的消息"
---

# Example Skill

这是一个示例技能，用于演示如何创建自定义技能。
```

3. 在 `main.go` 中添加对应的处理逻辑：

```go
// doExample 示例技能处理逻辑
func doExample(args map[string]interface{}) (string, error) {
	message, _ := args["message"].(string)
	return fmt.Sprintf("你输入的消息是: %s", message), nil
}

// 在 ExecuteSkill 函数中添加
if name == "example" {
	return doExample(args)
}
```

4. 运行 `reload_skills()` 重载技能

## 📡 API 依赖

| 服务 | 用途 | 来源 |
|------|------|------|
| 智谱 AI | LLM 调用 | https://open.bigmodel.cn/ |
| wttr.in | 天气查询 | https://wttr.in/ |
| ipapi.co | IP 定位 | https://ipapi.co/ |
| ipinfo.io | IP 定位 | https://ipinfo.io/ |
| MyMemory | 翻译服务 | https://api.mymemory.translated.net/ |

## 🔒 安全注意事项

- **API Key 安全**：不要将 API Key 提交到版本控制系统
- **命令执行**：Git 和系统信息技能有命令白名单限制
- **网络请求**：所有网络请求都有超时处理

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

**作者**：Your Name
**版本**：1.0.0
**最后更新**：2026-04-18