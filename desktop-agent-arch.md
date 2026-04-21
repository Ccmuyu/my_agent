# 桌面执行Agent架构文档

## 一、定位

让AI能够真正"操控"电脑，而不仅仅是聊天。

核心能力：
- 文件管理（读、写、重组织）
- 应用控制（操作桌面原生应用）
- 浏览器自动化（网页操作、表单填写）
- Shell命令执行
- 屏幕感知（截图+OCR）

---

## 二、技术选型

| 层级 | 技术选型 | 理由 |
|------|----------|------|
| **LLM** | OpenRouter API | 100+模型、支持Ollama、价格低 |
| **视觉感知** | pytesseract + opencv | 轻量级屏幕理解 |
| **桌面控制** | PyAutoGUI | 跨平台标准库 |
| **浏览器** | Playwright | 比Selenium更稳定 |
| **API层** | FastAPI + asyncio | 高并发、易部署 |
| **协议** | MCP | 标准化工具扩展 |

---

## 三、架构设计

```
┌─────────────────────────────────────────────────────────┐
│                     User Interface                      │
│              (CLI / Web UI / System Tray)               │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                   Task Engine                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │ Intent Parse│  │ Task Plan   │  │  Exec Loop  │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
│       │                │                │              │
│  ┌────▼────────────────▼────────────────▼────┐          │
│  │            State Machine                  │          │
│  │  PENDING → RUNNING → CONFIRM → COMPLETE   │          │
│  └───────────────────────────────────────────┘          │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                   Tool Layer (MCP Plugins)              │
│  ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐     │
│  │ File  │ │Browser│ │ App   │ │ Shell │ │ OCR   │     │
│  └───────┘ └───────┘ └───────┘ └───────┘ └───────┘     │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                   OS Layer                               │
│   Screenshot / Input Events / Window Management         │
└─────────────────────────────────────────────────────────┘
```

---

## 四、核心模块

### 1. Perception（感知）
```python
class ScreenPerception:
    def capture(self) -> PIL.Image: ...
    def OCR(self, image, region=None) -> str: ...
    def locate_element(self, template) -> tuple: ...
    def get_window_info(self) -> dict: ...
```

### 2. Planning（规划）
```python
class TaskPlanner:
    def parse(self, user_input: str) -> TaskIntent: ...
    def plan(self, intent: TaskIntent) -> list[Action]: ...
    def validate(self, actions: list[Action]) -> bool: ...
```

### 3. Execution（执行）
```python
class ActionExecutor:
    async def execute(self, action: Action) -> Result: ...
    async def retry(self, action: Action, max_retries=3) -> Result: ...
    def rollback(self, action: Action) -> None: ...
```

### 4. Tool Registry
```python
class ToolRegistry:
    def register(self, name: str, func: callable): ...
    def list_tools(self) -> list[str]: ...
    def call(self, name: str, **kwargs) -> Any: ...
```

---

## 五、工具集

| 工具 | 能力 |
|------|------|
| `file_read` | 读取本地文件 |
| `file_write` | 写入/创建文件 |
| `file_list` | 列目录文件 |
| `file_rename` | 重命名/移动 |
| `browser_open` | 打开URL |
| `browser_click` | 点击元素 |
| `browser_input` | 填写表单 |
| `browser_screenshot` | 网页截图 |
| `app_launch` | 启动应用 |
| `app_control` | 控制应用 |
| `shell_run` | 执行命令 |
| `screen_capture` | 屏幕截图 |
| `ocr_extract` | 文字识别 |

---

## 六、MCP协议

支持MCP扩展工具生态：

```python
from mcp import Tool

@Tool(name="custom_tool", description="...")
async def custom_tool(param: str) -> str:
    """自定义MCP工具"""
    return result
```

---

## 七、部署方案

### 方案A：纯本地（推荐）
```
┌─────────────┐
│   Desktop   │
│  Agent UI   │
├─────────────┤
│ Tool Layer │
├─────────────┤
│  PyAutoGUI  │
│  Playwright│
└─────────────┘
     ↑
  Ollama (可选本地模型)
```
- 成本：$0
- 隐私：⭐⭐⭐⭐⭐

### 方案B：云API
```
┌─────────────┐
│   Desktop   │
│  Agent UI   │
├─────────────┤
│ Tool Layer │
├─────────────┤
│  PyAutoGUI  │
│  Playwright│
└─────────────┘
     ↑
  OpenRouter API
```
- 成本：~$5/月
- 隐私：⭐⭐

### 方案C：混合模式
- 敏感操作本地执行
- 复杂理解用云API

---

## 八、开发路线

### Phase 1：文件Agent（Week 1-2）
- [ ] 文件读写
- [ ] 目录遍历
- [ ] 批量重命名

### Phase 2：浏览器Agent（Week 2-3）
- [ ] Playwright集成
- [ ] 网页操作
- [ ] 表单填写

### Phase 3：桌面Agent（Week 3-4）
- [ ] 屏幕截图+OCR
- [ ] 鼠标/键盘控制
- [ ] 应用启动

### Phase 4：编排（Week 4-5）
- [ ] 任务队列
- [ ] 定时执行
- [ ] MCP扩展

---

## 九、代码结构

```
desktop-agent/
├── agent/
│   ├── __init__.py
│   ├── core.py           # 核心引擎
│   ├── perception.py    # 感知模块
│   ├── planning.py     # 规划模块
│   ├── execution.py   # 执行模块
│   └── state.py       # 状态机
├── tools/
│   ├── __init__.py
│   ├── file.py        # 文件工具
│   ├── browser.py    # 浏览器工具
│   ├── app.py       # 应用工具
│   └── system.py    # 系统工具
├── mcp/
│   └── plugins/     # MCP插件
├── api/
│   └── main.py     # FastAPI入口
├── ui/              # Web UI
├── config.yaml
├── requirements.txt
└── main.py
```

---

## 十、API接口

```python
# 任务提交
POST /api/tasks
{
    "intent": "打开浏览器搜索AI新闻",
    "confirm": false  // 是否需要确认高风险操作
}

# 任务状态
GET /api/tasks/{task_id}

# 工具列表
GET /api/tools

# 屏幕截图
GET /api/screen
```

---

## 十一、注意事项

1. **安全限制**：删除/格式化等高风险操作需confirm
2. **速度瓶颈**：每个动作2-5秒，适合批量任务
3. **异常处理**：每个action需有rollback
4. **隐私**：敏感操作可在本地执行

---

要我生成具体代码框架吗？