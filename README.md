# Dynamic Skill Agent

一个基于 Go 语言和多种大模型 API 的动态技能 Agent，支持多种实用技能如天气查询、翻译、系统信息等。

## ✨ 功能特性

- **多 Provider 支持**：支持智谱AI、阿里云百炼、火山引擎豆包、讯飞星火等多种大模型
- **动态技能管理**：自动加载 `skills` 目录下的技能定义
- **天气查询**：支持自动 IP 定位和手动指定城市，提供当前和未来天气
- **翻译功能**：支持多语言互译
- **系统信息**：查看系统资源使用情况
- **Git 操作**：执行 Git 相关命令
- **技能重载**：无需重启即可加载新技能
- **RAG 知识库**：基于向量数据库的语义搜索能力

## 🚀 快速开始

### 环境要求
- Go 1.18+

### 安装

```bash
git clone https://github.com/yourusername/dynamic-skill-agent.git
cd dynamic-skill-agent
go mod tidy
```

### 运行

```bash
# 默认使用智谱AI（需要设置 API Key）
go run main.go -provider=zhipu -api-key=YOUR_ZHIPU_API_KEY

# 使用阿里云百炼
go run main.go -provider=qwen -api-key=YOUR_QWEN_API_KEY

# 使用火山引擎豆包
go run main.go -provider=doubao -api-key=YOUR_DOUBAO_API_KEY

# 使用讯飞星火
go run main.go -provider=spark -api-key=YOUR_SPARK_API_KEY
```

## 📡 支持的大模型 Provider

| Provider | 提供商 | 默认模型 | 免费额度 | Base URL |
|----------|--------|----------|----------|----------|
| `zhipu` | 智谱AI | glm-4-flash | 2000万 Tokens（永久） | https://open.bigmodel.cn/api/paas/v4/ |
| `qwen` | 阿里云百炼 | qwen-plus | 100万 Tokens/模型（90天），Qwen-Turbo 每月100万（永久） | https://dashscope.aliyuncs.com/compatible-mode/v1 |
| `doubao` | 火山引擎豆包 | Doubao-Seed-2.0-lite | 5000万 Tokens（永久） | https://ark.cn-beijing.volces.com/api/v3 |
| `spark` | 讯飞星火 | spark-lite | 星火Lite 永久免费 | https://spark-api.xf-yun.com/v3.1/chat |

### 获取 API Key

**智谱AI**：
1. 访问 [智谱AI开放平台](https://open.bigmodel.cn/)
2. 注册并完成实名认证
3. 在控制台创建 API Key

**阿里云百炼**：
1. 访问 [阿里云百炼平台](https://bailian.console.aliyun.com/)
2. 注册并完成实名认证
3. 开通大模型服务并创建 API Key

**火山引擎豆包**：
1. 访问 [火山方舟](https://www.volcengine.com/product/ark)
2. 注册并完成实名认证
3. 创建推理接入点并获取 API Key

**讯飞星火**：
1. 访问 [讯飞开放平台](https://www.xfyun.cn/)
2. 注册并完成实名认证
3. 在控制台创建应用并获取 APPID、APIKey、APISecret

## 📚 技能列表

| 技能名称 | 描述 | 参数 | 示例 |
|---------|------|------|------|
| `weather` | 查询天气状态 | `city` (可选) | `weather(city: "北京")` |
| `translate_text` | 翻译文本 | `text`, `target_lang`, `source_lang` (可选) | `translate_text(text: "Hello", target_lang: "zh")` |
| `system_info` | 查看系统信息 | `command` | `system_info(command: "df -h")` |
| `git_operations` | 执行 Git 命令 | `command`, `path` (可选) | `git_operations(command: "status")` |
| `reload_skills` | 重载技能 | 无 | `reload_skills()` |

## 📚 RAG 知识库

支持基于 Qdrant 向量数据库的语义搜索，可用于构建私有知识库。

### 配置

在 `config.yaml` 中配置 RAG：

```yaml
rag:
  enabled: true
  vector_db:
    type: "qdrant"
    host: "localhost"
    port: 6333
    collection: "desktop-agent-kb"
  embedder:
    provider: "ollama"        # 或 "openai"
    model: "nomic-embed-text" # embedding 模型
    base_url: "http://localhost:11434"
  chunk:
    size: 512                 # 分块大小
    overlap: 50               # 分块重叠
  search:
    top_k: 5                  # 返回结果数
```

### 启动 Qdrant

```bash
# 使用 Docker 启动 Qdrant
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### 启动 Ollama Embedding 服务

```bash
# 启动 Ollama
ollama serve

# 拉取 embedding 模型
ollama pull nomic-embed-text
```

### RAG 工具

| 工具 | 参数 | 描述 |
|------|------|------|
| `rag_search` | `query`, `top_k` | 语义搜索知识库 |
| `rag_add` | `paths` | 添加文档到知识库 |
| `rag_list` | - | 列出已索引文档 |
| `rag_delete` | `document_id` | 删除文档 |

### 使用示例

```go
// 1. 从配置创建 RAG 服务
service, err := rag.NewRAGServiceFromConfig(ctx, cfg)

// 2. 注册到工具注册表
ragTool := tools.NewRAGTool(service, ctx)
ragTool.RegisterToRegistry(registry)

// 3. 调用工具
result, err := registry.Call("rag_search", map[string]any{
    "query": "如何启动应用",
    "top_k": 3,
})

// 添加文档到知识库
result, err := registry.Call("rag_add", map[string]any{
    "paths": []string{"/path/to/doc1.md", "/path/to/doc2.txt"},
})

// 列出已索引文档
result, err := registry.Call("rag_list", map[string]any{})
```

### 支持的文件格式

- 文本文件：`.txt`, `.md`
- 代码文件：`.go`, `.py`, `.js`, `.ts`, `.java`, `.c`, `.cpp` 等
- 配置文件：`.json`, `.yaml`, `.yml`, `.xml`
- 其他：`.html`, `.css`, `.sql`, `.sh`

> 注意：PDF 支持需要使用外部工具（如 pdftotext）预处理

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

## 📡 外部 API 依赖

| 服务 | 用途 | 来源 |
|------|------|------|
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
**版本**：2.0.0
**最后更新**：2026-04-18