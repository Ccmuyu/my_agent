# RAG 向量数据库规划

## 一、方案概述

| 组件 | 推荐选型 | 理由 |
|------|----------|------|
| **向量数据库** | Qdrant | Rust开发、高性能、支持本地、Go客户端成熟 |
| **embedding模型** | 本地部署 `bge-m3` 或 `nomic-embed-text` | 支持本地、保护隐私 |
| **文档处理** | 支持PDF/MD/TXT/代码 | 解析+分块+向量化 |

## 二、目标场景

1. **知识库问答** - 让Agent能回答关于系统/项目的问题
2. **文档检索** - 基于语义检索相关文档
3. **历史经验** - 复用之前类似任务的成功经验

## 三、架构设计

```
┌─────────────────────────────────────────┐
│               RAG Service               │
│  ┌─────────────┐  ┌─────────────────┐  │
│  │ Document    │  │ Query           │  │
│  │ Ingestion   │  │ Processing      │  │
│  └──────┬──────┘  └────────┬────────┘  │
│         │                  │            │
│  ┌──────▼──────────────────▼──────┐    │
│  │       Vector Store (Qdrant)    │    │
│  └──────────────────┬─────────────┘    │
└─────────────────────┼──────────────────┘
                      │
┌─────────────────────▼──────────────────┐
│         Agent Tool Layer                │
│  ┌─────────────────────────────────┐   │
│  │ rag_search(query) → context     │   │
│  │ rag_add_documents(paths)        │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

## 四、核心模块设计

### 1. 新增目录结构

```
internal/
├── rag/
│   ├── client.go        # Qdrant客户端封装
│   ├── document.go      # 文档处理、分块
│   ├── embedder.go      # embedding模型调用
│   ├── service.go       # RAG核心服务
│   └── tool.go          # MCP工具注册
```

### 2. 核心接口

```go
// 文档 ingestion
type DocumentIngester interface {
    Process(ctx context.Context, paths []string) error
    Chunk(text string, chunkSize int) []string
}

// 向量存储
type VectorStore interface {
    Insert(ctx context.Context, vectors []Vector, payloads []Payload) error
    Search(ctx context.Context, query Vector, topK int) ([]SearchResult, error)
}

// RAG服务
type RAGService interface {
    IndexDocuments(ctx context.Context, docPaths []string) error
    Search(ctx context.Context, query string, topK int) (string, error)
}
```

### 3. 新增Tool

| Tool | 描述 |
|------|------|
| `rag_search` | 语义搜索知识库，返回相关上下文 |
| `rag_add` | 添加文档到知识库 |
| `rag_list` | 列出已索引的文档 |
| `rag_delete` | 删除指定文档 |

## 五、配置扩展

```yaml
rag:
  enabled: true
  vector_db:
    type: "qdrant"
    host: "localhost"
    port: 6333
    collection: "desktop-agent-kb"
  embedder:
    provider: "ollama"
    model: "nomic-embed-text"
    base_url: "http://localhost:11434"
  chunk:
    size: 512
    overlap: 50
  search:
    top_k: 5
```

## 六、实施步骤

| 阶段 | 任务 | 预估 |
|------|------|------|
| **Phase 1** | 集成Qdrant客户端，搭建基础服务 | 1天 |
| **Phase 2** | 实现文档解析+分块+向量化 | 2天 |
| **Phase 3** | 实现语义搜索+上下文注入 | 1天 |
| **Phase 4** | 注册MCP工具，集成到Agent | 1天 |

## 七、依赖

```go
// go.mod 新增
github.com/qdrant/go-client v1.8.0
github.com/tmc/langchaingo v0.1.0  // embedding支持
```

---

## 阶段实施记录

### Phase 1: Qdrant客户端集成
- [x] 配置扩展（config.go, config.yaml）
- [x] Qdrant客户端封装（client.go）
- [x] Embedder封装（embedder.go）
- [x] RAG服务基础（service.go）
- [x] 单元测试（service_test.go） 

### Phase 2: 文档处理
- [x] 文档解析（document.go）- 支持txt/md/代码等
- [x] 分块函数（service.go）- ChunkText
- [x] Embedder封装（embedder.go）- Ollama/OpenAI
- [x] 单元测试（service_test.go, embedder_test.go） 

### Phase 3: 语义搜索
- [x] RAG服务Search方法（service.go）
- [x] SearchWithSources返回源信息（service.go）
- [x] ListDocuments/DeleteDocument（service.go）
- [x] 工厂函数（factory.go）
- [x] 单元测试 

### Phase 4: MCP工具集成
- [x] RAG工具类（tools/rag.go）
- [x] 注册到ToolRegistry（rag.go）
- [x] 4个工具：rag_search, rag_add, rag_list, rag_delete