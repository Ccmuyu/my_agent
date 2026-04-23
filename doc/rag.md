# RAG 知识库

支持基于 Qdrant 向量数据库的语义搜索，可用于构建私有知识库。

## 配置

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

## 启动 Qdrant

```bash
# 使用 Docker 启动 Qdrant
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

## 启动 Ollama Embedding 服务

```bash
# 启动 Ollama
ollama serve

# 拉取 embedding 模型
ollama pull nomic-embed-text
```

## RAG 工具

| 工具 | 参数 | 描述 |
|------|------|------|
| `rag_search` | `query`, `top_k` | 语义搜索知识库 |
| `rag_add` | `paths` | 添加文档到知识库 |
| `rag_list` | - | 列出已索引文档 |
| `rag_delete` | `document_id` | 删除文档 |

## 使用示例

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

## 支持的文件格式

- 文本文件：`.txt`, `.md`
- 代码文件：`.go`, `.py`, `.js`, `.ts`, `.java`, `.c`, `.cpp` 等
- 配置文件：`.json`, `.yaml`, `.yml`, `.xml`
- 其他：`.html`, `.css`, `.sql`, `.sh`

> 注意：PDF 支持需要使用外部工具（如 pdftotext）预处理