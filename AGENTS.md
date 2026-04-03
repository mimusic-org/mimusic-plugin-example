# MiMusic 插件开发规范

本文档详细说明了如何为 MiMusic 音乐服务器开发 WebAssembly 插件。

## 目录

- [快速开始](#快速开始)
- [插件架构](#插件架构)
- [项目结构](#项目结构)
- [开发步骤](#开发步骤)
- [核心 API 使用](#核心-api-使用)
- [数据持久化](#数据持久化)
- [代码规范](#代码规范)
- [最佳实践](#最佳实践)
- [调试与测试](#调试与测试)
- [发布与部署](#发布与部署)

## 快速开始

### 环境要求

- Go 1.24+
- 支持 WASI 的 Go 工具链
- Make（可选，用于构建自动化）

### 重要：构建参数

**1. 构建时必须添加 `-buildmode=c-shared` 参数**，否则插件会因运行时未初始化而失败：

```bash
# 正确的构建命令
GOOS=wasip1 GOARCH=wasm go build -o plugin.wasm -buildmode=c-shared .

# 错误的构建命令（会导致 runtime error）
GOOS=wasip1 GOARCH=wasm go build -o plugin.wasm .
```

**2. 静态资源和 API 路径规范**

**路由注册**时使用 `EntryPath`（如 `/myplugin`），**前端代码使用**时需要添加 `/api/v1/plugin/` 前缀：

```go
// 路由注册（使用 EntryPath，不需要 /api/v1/plugin/ 前缀）
rm := plugin.GetRouterManager()

// 使用 NewStaticHandler 自动注册静态资源
p.staticHandler = plugin.NewStaticHandler(staticFS, "/myplugin", rm, ctx)

// 注册其他业务路由
rm.RegisterRouter(ctx, "GET", "/myplugin", p.handleIndex, false)
rm.RegisterRouter(ctx, "GET", "/myplugin/api/data", p.apiHandler.HandleGetData, true)
```

```javascript
// 前端代码（需要 /api/v1/plugin/ 前缀）
// ✓ 正确的路径
<link rel="stylesheet" href="/api/v1/plugin/myplugin/static/css/style.css">
<script src="/api/v1/plugin/myplugin/static/js/app.js"></script>
fetch('/api/v1/plugin/myplugin/api/data')
fetch('/api/v1/plugin/myplugin/api/submit', { method: 'POST' })

// ✗ 错误的路径（缺少 /api/v1/plugin/ 前缀）
<link rel="stylesheet" href="/myplugin/static/css/style.css">
<script src="/myplugin/static/js/app.js"></script>
fetch('/myplugin/api/data')
```

**原因**：宿主框架会自动将 `/api/v1/plugin/{plugin_name}/` 前缀映射到插件的路由，所以插件注册时使用 `EntryPath` 即可。

**3. 所有 Go 文件必须添加构建标签**

每个 Go 文件的开头都必须添加以下构建标签，以确保正确的编译目标：

```go
//go:build wasip1
// +build wasip1

package main
```

### 创建新插件

1. **使用示例模板**

   访问 [mimusic-plugin-example](https://github.com/mimusic-org/mimusic-plugin-example) 使用该模板创建新仓库，或直接克隆：
   ```bash
   git clone https://github.com/mimusic-org/mimusic-plugin-example mimusic-plugin-myplugin
   ```

2. **修改基础配置**
   - 更新 `main.go` 中的插件元数据
   - 修改 `Makefile` 中的 `PLUGIN_NAME`

3. **安装依赖**
   ```bash
   cd mimusic-plugin-myplugin
   go mod init mimusic-plugin-myplugin
   go get github.com/mimusic-org/plugin
   ```

## 插件架构

### 生命周期

插件遵循标准的生命周期模式：

```
注册 → 初始化 → 运行 → 反初始化
  ↓        ↓        ↓        ↓
init()  Init()   处理请求  Deinit()
```

### 核心组件

```go
// 插件结构体示例
type Plugin struct {
    // 业务管理器
    accountManager *account.Manager
    authService    *auth.Service
    
    // HTTP 处理器
    staticHandler   *handlers.StaticHandler
    apiHandler      *handlers.APIHandler
}
```

### 注册机制

```go
// init 函数自动注册插件
func init() {
    plugin.RegisterPlugin(&Plugin{})
}
```

## 项目结构

### 标准目录结构

```
mimusic-plugin-myplugin/
├── main.go                 # 插件入口和生命周期实现
├── Makefile               # 构建脚本
├── go.mod                 # Go 模块定义
├── go.sum                 # 依赖锁定
├── account/               # 业务模块
│   ├── manager.go
│   └── types.go
├── auth/                  # 认证模块
│   ├── service.go
│   └── captcha.go
├── handlers/              # HTTP 处理器
│   ├── static.go
│   ├── account.go
│   └── api.go
└── static/                # 静态资源
    ├── css/
    ├── js/
    └── images/
```

### 文件职责说明

| 文件/目录 | 职责 | 必需 |
|----------|------|------|
| `main.go` | 插件入口、生命周期、路由注册 | ✓ |
| `Makefile` | 构建自动化 | 推荐 |
| `handlers/` | HTTP 请求处理 | ✓ |
| 业务模块 | 核心业务逻辑 | ✓ |

## 前端 UI 规范

插件的 Web 前端界面应遵循 Material Design 3 风格，与 MiMusic Flutter 客户端保持视觉一致。

### 色彩系统

使用 CSS 变量定义 Material Design 3 调色板。主色从 Flutter 客户端的 `ColorScheme.fromSeed(seedColor: Color(0xFF6366F1))` 生成。

```css
:root {
    /* 主色 - 与 Flutter 客户端 seedColor indigo 一致 */
    --md-primary: #595b94;        /* Primary */
    --md-on-primary: #FFFFFF;
    --md-primary-container: #E8DEF8;
    --md-on-primary-container: #21005E;

    /* Surface */
    --md-surface: #FFFBFE;
    --md-on-surface: #1C1B1F;
    --md-on-surface-variant: #49454F;
    --md-surface-variant: #E7E0EC;

    /* Outline */
    --md-outline: #79747E;
    --md-outline-variant: #CAC4D0;

    /* 语义色 */
    --md-error: #B3261E;
    --md-on-error: #FFFFFF;
    --md-error-container: #F9DEDC;
    --md-success: #2E7D32;
    --md-success-container: #C8E6C9;
    --md-warning: #E65100;
    --md-warning-container: #FFE0B2;

    /* Elevation */
    --md-surface-1: #F0EEF8;
    --md-surface-2: #E8E6F2;
    --md-shadow-1: 0 1px 2px rgba(0,0,0,.12), 0 1px 3px rgba(0,0,0,.08);
    --md-shadow-2: 0 2px 4px rgba(0,0,0,.14), 0 1px 6px rgba(0,0,0,.1);

    /* 圆角 */
    --md-radius-sm: 4px;
    --md-radius-md: 12px;
    --md-radius-lg: 16px;
    --md-radius-xl: 20px;
    --md-radius-full: 50px;
}
```

### 字体

使用本地化字体（打包到插件 `static/fonts/` 目录），避免依赖 CDN：

```css
/* static/css/fonts.css */
@font-face {
    font-family: 'Roboto';
    src: url('/api/v1/plugin/{plugin_name}/static/fonts/roboto-400.woff2') format('woff2');
    font-weight: 400;
    font-display: swap;
}
/* Material Symbols Outlined 图标字体同样本地化 */
```

字体族顺序：`'Roboto', 'Noto Sans SC', system-ui, sans-serif`

### 核心组件

插件前端应使用以下 Material Design 3 组件样式（纯 CSS 实现，无框架依赖）：

| 组件 | CSS 类名 | 说明 |
|------|----------|------|
| AppBar | `.app-bar` | 固定顶部，主色背景 |
| Card | `.card` | 圆角 12px，elevation 阴影 |
| Filled Button | `.btn-filled` | 主色背景，圆角 20px |
| Outlined Button | `.btn-outlined` | 主色边框，透明背景 |
| Text Button | `.btn-text` | 无边框，主色文字 |
| Icon Button | `.btn-icon` | 圆形按钮 |
| TextField | `.text-field` | Material 风格输入框 |
| Select | `.select-field` | 下拉选择框 |
| Switch | `.md-switch` | Material 开关 |
| Checkbox | 原生 + `accent-color` | 使用主色 |
| Snackbar | `.snackbar` | 底部提示，替代 alert/toast |
| Dialog | `.dialog-overlay` + `.dialog` | 模态对话框，替代 confirm() |
| Progress | `.progress-linear` | 线性进度条 |
| Tab Bar | `.tab-bar` | 底部 Tab 导航（匹配 Flutter NavigationBar） |

### 布局规范

- **Tab 导航**：当插件有多个功能模块时，使用底部 Tab Bar（固定底部，64px 高度）
- **响应式断点**：600px（移动）/ 900px（平板）/ 1920px+（TV），与 Flutter 客户端一致
- **内容容器**：`max-width: 960px`，水平居中
- **卡片间距**：16px gap

### 认证机制

插件前端从 `localStorage` 获取主程序的认证令牌：

```javascript
function getAuthToken() {
    try {
        const authData = localStorage.getItem('mimusic-auth');
        if (authData) return JSON.parse(authData).accessToken || '';
    } catch (e) {}
    return '';
}

// 所有 API 请求携带 Authorization header
headers['Authorization'] = 'Bearer ' + getAuthToken();
```

### 参考实现

完整的 Material Design 3 前端实现请参考 [mimusic-plugin-lxmusic](https://github.com/mimusic-org/mimusic-plugins/tree/main/mimusic-plugin-lxmusic) 插件：
- `static/css/style.css` — 完整的 Material Design 3 组件样式
- `static/css/fonts.css` — 本地字体声明
- `static/fonts/` — 本地化字体文件（Roboto + Material Symbols）
- `static/js/app.js` — 前端功能逻辑
- `static/index.html` — 页面布局

## 开发步骤

### 步骤 1: 定义插件元数据

在 `GetPluginInfo` 方法中返回插件信息：

```go
func (p *Plugin) GetPluginInfo(ctx context.Context, request *emptypb.Empty) (*pbplugin.GetPluginInfoResponse, error) {
    return &pbplugin.GetPluginInfoResponse{
        Success: true,
        Message: "成功获取插件信息",
        Info: &pbplugin.PluginInfo{
            Name:        "我的插件",           // 插件显示名称
            Version:     "1.0.0",            // 语义化版本号
            Description: "插件功能描述",       // 简短描述
            Author:      "作者名",            // 作者信息
            Homepage:    "https://...",      // 项目主页
            EntryPath:   "/myplugin",        // 路由前缀
        },
    }, nil
}
```

**命名规范**：
- `Name`: 中文名称，2-10 个字符
- `Version`: 遵循语义化版本 (MAJOR.MINOR.PATCH)
- `EntryPath`: 小写字母，以 `/` 开头，无尾随斜杠

### 步骤 2: 实现初始化逻辑

在 `Init` 方法中完成初始化和路由注册：

```go
func (p *Plugin) Init(ctx context.Context, request *pbplugin.InitRequest) (*emptypb.Empty, error) {
    slog.Info("正在初始化插件", "version", "1.0.0")
    
    // 1. 初始化管理器和服务
    p.accountManager, err = account.NewManager("/myplugin")
    if err != nil {
        return &emptypb.Empty{}, fmt.Errorf("failed to create manager: %w", err)
    }
    
    // 2. 初始化处理器
    p.apiHandler = handlers.NewAPIHandler(p.accountManager)
    
    // 3. 注册路由
    rm := plugin.GetRouterManager()
    
    // 使用 NewStaticHandler 自动注册静态资源
    p.staticHandler = plugin.NewStaticHandler(staticFS, "/myplugin", rm, ctx)
    
    // API 接口
    rm.RegisterRouter(ctx, "GET", "/myplugin/api/data", p.apiHandler.HandleGetData, true)
    rm.RegisterRouter(ctx, "POST", "/myplugin/api/submit", p.apiHandler.HandleSubmit, true)
    
    // 前端页面
    rm.RegisterRouter(ctx, "GET", "/myplugin", p.handleIndex, false)
    
    slog.Info("插件路由注册完成")
    return &emptypb.Empty{}, nil
}
```

### 步骤 3: 实现 HTTP 处理器

创建符合规范的处理器函数：

```go
// HandleGetData 处理获取数据请求
func (h *Handler) HandleGetData(req *http.Request) (*plugin.RouterResponse, error) {
    // 1. 解析请求参数
    query := req.URL.Query()
    id := query.Get("id")
    
    // 2. 业务逻辑处理
    data, err := h.service.GetData(id)
    if err != nil {
        return &plugin.RouterResponse{
            StatusCode: 400,
            Headers:    map[string]string{"Content-Type": "application/json"},
            Body:       []byte(`{"success":false,"message":"` + err.Error() + `"}`),
        }, nil
    }
    
    // 3. 返回响应
    return &plugin.RouterResponse{
        StatusCode: 200,
        Headers:    map[string]string{"Content-Type": "application/json"},
        Body:       data,
    }, nil
}
```

### 步骤 4: 实现清理逻辑

在 `Deinit` 方法中释放资源：

```go
func (p *Plugin) Deinit(ctx context.Context, request *emptypb.Empty) (*emptypb.Empty, error) {
    slog.Info("正在反初始化插件")
    
    // 关闭数据库连接、停止后台任务等
    if p.accountManager != nil {
        p.accountManager.Close()
    }
    
    return &emptypb.Empty{}, nil
}
```

## 核心 API 使用

### 路由管理

#### 注册路由

```go
rm := plugin.GetRouterManager()

// 注册简单路由
routeID := rm.RegisterRouter(ctx, "GET", "/path", handlerFunc, false)

// 注册带参数路由（需手动解析）
rm.RegisterRouter(ctx, "GET", "/users/{id}", handlerFunc, true)
```

**参数说明**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `ctx` | `context.Context` | 上下文对象 |
| `method` | `string` | HTTP 方法（GET/POST/PUT/DELETE 等） |
| `path` | `string` | 路由路径，以 EntryPath 为前缀 |
| `handler` | `func(*http.Request) (*plugin.RouterResponse, error)` | 请求处理函数 |
| `requireAuth` | `bool` | **是否需要认证**。`true` 表示需要用户登录才能访问，`false` 表示公开访问 |

**认证建议**：
- **静态资源**：设置为 `false`（CSS、JS、图片等不需要认证）
- **前端页面**：根据需求设置，通常首页设为 `false`
- **API 接口**：通常设为 `true`，需要用户认证后才能调用

**路由命名规范**：
- 使用小写字母和连字符
- 以插件入口路径为前缀
- RESTful 风格设计

```
✓ /myplugin/api/users
✓ /myplugin/api/users/{id}
✓ /myplugin/static/css/main.css

✗ /api/users              // 缺少插件前缀
✗ /MyPlugin/Users         // 大写不规范
```

#### 处理路由请求

```go
func (p *Plugin) handleRequest(req *http.Request) (*plugin.RouterResponse, error) {
    // 支持的方法
    switch req.Method {
    case http.MethodGet:
        return p.handleGet(req)
    case http.MethodPost:
        return p.handlePost(req)
    case http.MethodPut:
        return p.handlePut(req)
    case http.MethodDelete:
        return p.handleDelete(req)
    default:
        return &plugin.RouterResponse{
            StatusCode: http.StatusMethodNotAllowed,
            Body:       []byte("Method not allowed"),
        }, nil
    }
}
```

### 定时器管理

#### 注册定时器

```go
tm := plugin.GetTimerManager()

// 注册延迟定时器（参数单位：毫秒）
// 5000 毫秒 = 5 秒
timerID := tm.RegisterDelayTimer(ctx, 5000, func() {
    slog.Info("定时器触发")
    // 执行定时任务
})

// 返回的 timerID 可用于取消定时器
_ = timerID
```

#### 取消定时器

`RegisterDelayTimer` 函数会返回定时器ID，可用于取消已注册的定时器：

```go
// 保存定时器 ID
var timerID uint64

// 注册定时器并获取 ID
timerID = tm.RegisterDelayTimer(ctx, 5000, func() {
    slog.Info("定时器触发")
    // 执行定时任务
})

// 取消定时器
err := tm.CancelTimer(ctx, timerID)
if err != nil {
    slog.Warn("取消定时器失败", "error", err)
}
```

### 响应辅助函数

插件框架提供了三个便捷的响应辅助函数，简化常见响应的创建：

```go
import "github.com/mimusic-org/plugin/api/plugin"

// JSONResponse 创建 JSON 格式的 HTTP 响应
func JSONResponse(statusCode int, data interface{}) *RouterResponse

// ErrorResponse 创建错误响应（自动包含 success: false）
func ErrorResponse(statusCode int, message string) *RouterResponse

// SuccessResponse 创建成功响应（自动包含 success: true）
func SuccessResponse(data interface{}) *RouterResponse
```

**使用示例**：

```go
// 成功响应
return plugin.SuccessResponse(map[string]string{
    "user_id": "123",
    "username": "john",
})

// 错误响应
return plugin.ErrorResponse(400, "参数错误")

// 自定义 JSON 响应
return plugin.JSONResponse(201, map[string]interface{}{
    "message": "创建成功",
    "id": "123",
})
```

### 日志记录

使用标准 `slog` 包：

```go
import "log/slog"

// 不同级别的日志
slog.Debug("调试信息", "key", "value")
slog.Info("一般信息", "pluginId", plugin.GetPluginId())
slog.Warn("警告信息", "error", err)
slog.Error("错误信息", "error", err, "stack", stack)
```

## 静态资源管理

使用 `plugin.NewStaticHandler` 自动注册所有静态资源，无需手动注册每个文件：

```go
// main.go 中直接定义 embed.FS 并使用 NewStaticHandler 自动注册
//go:build wasip1
// +build wasip1

package main

import (
    "context"
    "embed"
    "log/slog"

    "github.com/mimusic-org/plugin/api/pbplugin"
    "github.com/mimusic-org/plugin/api/plugin"
    "github.com/knqyf263/go-plugin/types/known/emptypb"
)

//go:embed static/*
var staticFS embed.FS

func (p *Plugin) Init(ctx context.Context, request *pbplugin.InitRequest) (*emptypb.Empty, error) {
    slog.Info("正在初始化插件")

    // 获取路由管理器
    rm := plugin.GetRouterManager()

    // 创建静态资源处理器（自动注册所有 static 目录下的文件）
    p.staticHandler = plugin.NewStaticHandler(staticFS, "/myplugin", rm, ctx)

    // 注册其他业务路由
    rm.RegisterRouter(ctx, "GET", "/myplugin", p.handleIndex, false)
    rm.RegisterRouter(ctx, "POST", "/myplugin/api/data", p.handleApiData, true)

    slog.Info("插件路由注册完成")
    return &emptypb.Empty{}, nil
}
```

**优势**：
- ✅ 无需手动注册每个静态文件路由
- ✅ 自动处理 CSS、JS、图片等常见 MIME 类型
- ✅ 支持子目录结构（如 `/static/css/style.css`、`/static/js/app.js`）
- ✅ 代码更简洁，易于维护

**注意事项**：
- 静态文件必须放在 `static/` 目录下
- 前端代码中引用静态资源时仍需使用 `/api/v1/plugin/{plugin_name}/static/...` 路径
- `NewStaticHandler` 第2个参数是路由前缀（EntryPath），不需要 `/api/v1/plugin/` 前缀
- `NewStaticHandler` 第3、4个参数分别是路由管理器和上下文

## 数据持久化

### WASM 文件系统挂载机制

主程序通过 wazero 的 FSConfig 将宿主机的 `data/plugins_data/` 目录挂载到 WASM 沙盒的根目录 `/`。这意味着：

- 插件内访问 `/{plugin_name}/` 即映射到宿主机的 `data/plugins_data/{plugin_name}/`
- 使用标准 Go 文件 I/O 操作：`os.ReadFile`、`os.WriteFile`、`os.MkdirAll`、`os.Remove` 等
- 无需引入额外依赖，直接使用 `os` 包即可

```go
import "os"

// 读取文件
data, err := os.ReadFile("/myplugin/config.json")

// 写入文件
err := os.WriteFile("/myplugin/config.json", data, 0644)

// 创建目录
err := os.MkdirAll("/myplugin/data", 0755)

// 删除文件
err := os.Remove("/myplugin/cache/temp.json")
```

### 目录结构规范

每个插件使用 `/{plugin_name}/` 作为自己的数据根目录，建议在 `Init()` 中确保目录存在：

```go
func (p *Plugin) Init(ctx context.Context, request *pbplugin.InitRequest) (*emptypb.Empty, error) {
    // 确保插件数据目录存在
    if err := os.MkdirAll("/myplugin", 0755); err != nil {
        return &emptypb.Empty{}, fmt.Errorf("failed to create data dir: %w", err)
    }
    
    // 初始化业务管理器，加载持久化数据
    p.manager, err = NewManager("/myplugin")
    if err != nil {
        return &emptypb.Empty{}, err
    }
    
    return &emptypb.Empty{}, nil
}
```

**推荐的目录结构**：

```
/{plugin_name}/
├── config.json           # 插件配置
├── data/                 # 业务数据目录
│   ├── index.json        # 数据索引
│   └── items/            # 具体数据文件
│       ├── item1.json
│       └── item2.json
└── cache/                # 临时缓存（可选）
```

### 持久化模式

#### 模式一：单 JSON 配置文件

适用于配置数据、账号信息等结构化配置。参考 mimusic-plugin-xiaomi 的实现：

```go
// types.go - 定义配置结构
type Config struct {
    Accounts []Account `json:"accounts"`
    Settings Settings  `json:"settings"`
}

type Account struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    Token    string `json:"token"`
}

// manager.go - 管理器实现
type Manager struct {
    dataDir    string
    configPath string
    config     *Config
}

func NewManager(dataDir string) (*Manager, error) {
    m := &Manager{
        dataDir:    dataDir,
        configPath: dataDir + "/config.json",
        config:     &Config{},
    }
    
    // 加载配置
    if err := m.loadConfig(); err != nil {
        slog.Warn("加载配置失败，使用默认配置", "error", err)
    }
    
    return m, nil
}

func (m *Manager) loadConfig() error {
    data, err := os.ReadFile(m.configPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil // 文件不存在，使用默认配置
        }
        return err
    }
    return json.Unmarshal(data, m.config)
}

func (m *Manager) saveConfig() error {
    data, err := json.MarshalIndent(m.config, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(m.configPath, data, 0644)
}

// 业务方法：添加账号后立即保存
func (m *Manager) AddAccount(account Account) error {
    m.config.Accounts = append(m.config.Accounts, account)
    return m.saveConfig() // 每次写操作后立即持久化
}
```

#### 模式二：索引 + 文件分离存储

适用于脚本文件、模板文件、用户上传的文件等大量独立文件。参考 mimusic-plugin-lxmusic 的实现：

```go
// types.go - 定义索引结构
type SourceIndex struct {
    Sources []SourceMeta `json:"sources"`
}

type SourceMeta struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Filename string `json:"filename"` // 实际文件名
    Enabled  bool   `json:"enabled"`
}

// manager.go - 管理器实现
type SourceManager struct {
    dataDir   string
    indexPath string
    sourcesDir string
    index     *SourceIndex
    scripts   map[string][]byte // 内存缓存
}

func NewSourceManager(dataDir string) (*SourceManager, error) {
    m := &SourceManager{
        dataDir:    dataDir,
        indexPath:  dataDir + "/index.json",
        sourcesDir: dataDir + "/sources",
        index:      &SourceIndex{},
        scripts:    make(map[string][]byte),
    }
    
    // 确保目录存在
    if err := os.MkdirAll(m.sourcesDir, 0755); err != nil {
        return nil, err
    }
    
    // 加载索引和文件
    if err := m.load(); err != nil {
        slog.Warn("加载数据失败", "error", err)
    }
    
    return m, nil
}

func (m *SourceManager) load() error {
    // 1. 加载索引文件
    data, err := os.ReadFile(m.indexPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }
    if err := json.Unmarshal(data, m.index); err != nil {
        return err
    }
    
    // 2. 根据索引加载各个文件
    for _, meta := range m.index.Sources {
        filePath := m.sourcesDir + "/" + meta.Filename
        content, err := os.ReadFile(filePath)
        if err != nil {
            slog.Warn("加载文件失败，跳过", "id", meta.ID, "error", err)
            continue // 优雅降级：跳过损坏文件
        }
        m.scripts[meta.ID] = content
    }
    
    return nil
}

func (m *SourceManager) saveIndex() error {
    data, err := json.MarshalIndent(m.index, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(m.indexPath, data, 0644)
}

// 添加新文件
func (m *SourceManager) AddSource(id, name string, content []byte) error {
    filename := id + ".js"
    filePath := m.sourcesDir + "/" + filename
    
    // 1. 保存文件内容
    if err := os.WriteFile(filePath, content, 0644); err != nil {
        return err
    }
    
    // 2. 更新索引
    m.index.Sources = append(m.index.Sources, SourceMeta{
        ID:       id,
        Name:     name,
        Filename: filename,
        Enabled:  true,
    })
    
    // 3. 保存索引
    if err := m.saveIndex(); err != nil {
        return err
    }
    
    // 4. 更新内存缓存
    m.scripts[id] = content
    
    return nil
}

// 删除文件
func (m *SourceManager) RemoveSource(id string) error {
    // 1. 从索引中查找并移除
    var filename string
    for i, meta := range m.index.Sources {
        if meta.ID == id {
            filename = meta.Filename
            m.index.Sources = append(m.index.Sources[:i], m.index.Sources[i+1:]...)
            break
        }
    }
    
    if filename == "" {
        return fmt.Errorf("source not found: %s", id)
    }
    
    // 2. 删除文件
    filePath := m.sourcesDir + "/" + filename
    if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
        slog.Warn("删除文件失败", "path", filePath, "error", err)
    }
    
    // 3. 保存索引
    if err := m.saveIndex(); err != nil {
        return err
    }
    
    // 4. 清除内存缓存
    delete(m.scripts, id)
    
    return nil
}
```

**目录结构示例**：

```
/lxmusic/
├── index.json            # 元数据索引
└── sources/              # 实际文件存储
    ├── source1.js
    ├── source2.js
    └── source3.js
```

### 最佳实践

1. **Init() 时加载数据到内存**
   ```go
   func (p *Plugin) Init(ctx context.Context, request *pbplugin.InitRequest) (*emptypb.Empty, error) {
       // 创建管理器时自动加载持久化数据
       p.manager, err = NewManager("/myplugin")
       if err != nil {
           return &emptypb.Empty{}, err
       }
       return &emptypb.Empty{}, nil
   }
   ```

2. **每次写操作后立即持久化**
   ```go
   // ✓ 推荐：修改后立即保存
   func (m *Manager) UpdateSetting(key, value string) error {
       m.config.Settings[key] = value
       return m.saveConfig() // 立即持久化
   }
   
   // ✗ 避免：等 Deinit 统一保存（可能丢失数据）
   ```

3. **文件加载失败时优雅降级**
   ```go
   for _, meta := range m.index.Sources {
       content, err := os.ReadFile(filePath)
       if err != nil {
           slog.Warn("加载文件失败，跳过", "id", meta.ID, "error", err)
           continue // 跳过损坏文件，继续加载其他文件
       }
       m.scripts[meta.ID] = content
   }
   ```

4. **使用 `json.MarshalIndent` 保持可读性**
   ```go
   // ✓ 推荐：格式化输出，便于调试
   data, err := json.MarshalIndent(config, "", "  ")
   
   // ✗ 避免：紧凑格式，难以阅读
   data, err := json.Marshal(config)
   ```

5. **无需使用文件锁**
   - WASM 环境是单线程执行的，不存在并发写入问题
   - 直接使用 `os.WriteFile` 即可，无需 `sync.Mutex` 或文件锁

## 代码规范

### 命名规范

#### 包命名
```go
// ✓ 推荐：小写，无下划线
package account
package auth
package handlers

// ✗ 避免：大写、下划线、中划线
package Account
package user_info
package my-handlers
```

#### 结构体命名
```go
// ✓ 推荐：名词，大驼峰
type UserManager struct {}
type AuthService struct {}
type Config struct {}

// ✗ 避免：动词、小写开头
type manageUser struct {}
type configData struct {}
```

#### 函数命名
```go
// ✓ 推荐：动词开头，大驼峰
func (s *Service) GetUserByID(id int64) (*User, error)
func (h *Handler) HandleLogin(req *http.Request) (*RouterResponse, error)

// ✗ 避免：名词开头、小写
func (s *Service) userGetter() {}
func (h *Handler) handle_login() {}
```

### 错误处理

```go
// ✓ 推荐：明确的错误处理
accountManager, err := account.NewManager("/myplugin")
if err != nil {
    slog.Error("创建管理器失败", "error", err)
    return &emptypb.Empty{}, fmt.Errorf("failed to create manager: %w", err)
}

// ✗ 避免：忽略错误
accountManager, _ := account.NewManager("/myplugin")
```

### 注释规范

```go
// GetPluginInfo 返回插件元数据
// 包含名称、版本、描述、作者和主页信息
func (p *Plugin) GetPluginInfo(ctx context.Context, request *emptypb.Empty) (*pbplugin.GetPluginInfoResponse, error) {
    // ...
}

// ✓ 推荐：解释"为什么"而非"怎么做"
// 使用内存缓存减少数据库查询
var cache = make(map[string]interface{})

// ✗ 避免：冗余注释
// 增加计数器
count++
```

### 代码组织

```go
// ✓ 推荐：按逻辑分组
type Plugin struct {
    // 管理器
    accountManager *account.Manager
    authService    *auth.Service
    
    // 处理器
    staticHandler   *handlers.StaticHandler
    apiHandler      *handlers.APIHandler
}

// 方法按生命周期排序
func (p *Plugin) GetPluginInfo() {}
func (p *Plugin) Init() {}
func (p *Plugin) Deinit() {}
func (p *Plugin) handleRequest() {}
```

### 7. 并发注意事项

**重要**：插件在 WASM 环境中是**单线程执行**的，因此：

- ✅ **不需要使用锁**（sync.Mutex、sync.RWMutex 等）
- ✅ 不需要使用 sync.Pool 等并发工具
- ✅ 不需要使用 atomic 操作
- ✗ **禁止使用 goroutine**（插件环境不支持）
- ✗ 后台任务请使用定时器实现

```go
// ✓ 正确：直接使用 map，无需锁
var cache = make(map[string]interface{})

func handleRequest() {
    cache["key"] = "value"
    // 单线程执行，无需加锁
}

// ✗ 错误：使用锁（不必要的开销）
var mu sync.RWMutex
var cache = make(map[string]interface{})

func handleRequest() {
    mu.Lock()
    cache["key"] = "value"
    mu.Unlock()
}
```

### 8. 性能优化
## 调试与测试

### 本地调试

```bash
# 构建 WASM 文件（必须添加 -buildmode=c-shared）
GOOS=wasip1 GOARCH=wasm go build -o ${PLUGIN_NAME}.wasm -buildmode=c-shared

# 或使用 Makefile
make build

# 查看插件信息
make info
```

### 日志调试

```go
// 开发阶段使用详细日志
if os.Getenv("DEBUG") == "true" {
    slog.SetLogLoggerLevel(slog.LevelDebug)
}

// 关键节点打点
slog.Debug("进入函数", "function", "HandleLogin")
slog.Debug("请求参数", "params", params)
slog.Debug("查询结果", "result", result)
```

### 错误排查

```go
// 完整的错误上下文
if err != nil {
    slog.Error("操作失败",
        "operation", "login",
        "account_id", accountID,
        "error", err,
        "stack", string(debug.Stack()),
    )
    return nil, err
}
```

## 发布与部署

### 版本管理

遵循语义化版本规范：

```
1.0.0  # 初始版本
1.0.1  # Bug 修复
1.1.0  # 新功能（向后兼容）
2.0.0  # 破坏性变更
```

### 构建发布

```bash
# 更新版本号（必须添加 -buildmode=c-shared）
VERSION=2.0.0 GOOS=wasip1 GOARCH=wasm go build -o myplugin.wasm -buildmode=c-shared

# 或使用 Makefile
VERSION=2.0.0 make build
```

### 上传插件

1. 通过 MiMusic 管理界面上传 `.wasm` 文件
2. 系统自动提取插件元数据
3. 启用插件并验证功能

### 更新插件

```bash
# 更新版本号
# main.go: Version: "2.0.0"

# 重新构建
make build

# 上传新版本
# 系统会自动替换旧版本
```

## 安全注意事项

### 1. 敏感信息保护

```go
// ✓ 推荐：从配置文件读取
config := loadConfig("/myplugin/config.json")
apiKey := config.APIKey

// ✗ 避免：硬编码在代码中
apiKey := "sk-1234567890abcdef"  // 危险！
```

### 2. 输入验证

```go
// 验证所有用户输入
func validateInput(input string) error {
    if len(input) > 1000 {
        return fmt.Errorf("input too long")
    }
    if !isValidUTF8(input) {
        return fmt.Errorf("invalid encoding")
    }
    return nil
}
```

### 3. 错误信息脱敏

```go
// ✓ 推荐：通用错误消息
if authFailed {
    return errors.New("认证失败")  // 不暴露具体原因
}

// ✗ 避免：暴露内部信息
if authFailed {
    return errors.New("密码错误，剩余尝试次数：2")  
}
```

## 示例代码模板

### 完整插件模板

```go
//go:build wasip1
// +build wasip1

package main

import (
    "context"
    "fmt"
    "log/slog"
    "net/http"

	"github.com/mimusic-org/plugin/api/pbplugin"
	"github.com/mimusic-org/plugin/api/plugin")

func main() {}

type Plugin struct {
    // 业务组件
}

func init() {
    plugin.RegisterPlugin(&Plugin{})
}

func (p *Plugin) GetPluginInfo(ctx context.Context, request *emptypb.Empty) (*pbplugin.GetPluginInfoResponse, error) {
    return &pbplugin.GetPluginInfoResponse{
        Success: true,
        Message: "成功获取插件信息",
        Info: &pbplugin.PluginInfo{
            Name:        "插件名称",
            Version:     "1.0.0",
            Description: "功能描述",
            Author:      "作者",
            Homepage:    "https://...",
            EntryPath:   "/plugin-path",
        },
    }, nil
}

func (p *Plugin) Init(ctx context.Context, request *pbplugin.InitRequest) (*emptypb.Empty, error) {
    slog.Info("正在初始化插件")
    
    rm := plugin.GetRouterManager()
    
    // 注册路由
    rm.RegisterRouter(ctx, "GET", "/plugin-path", p.handleIndex, false)
    
    slog.Info("插件初始化完成")
    return &emptypb.Empty{}, nil
}

func (p *Plugin) Deinit(ctx context.Context, request *emptypb.Empty) (*emptypb.Empty, error) {
    slog.Info("正在反初始化插件")
    return &emptypb.Empty{}, nil
}

func (p *Plugin) handleIndex(req *http.Request) (*plugin.RouterResponse, error) {
    return &plugin.RouterResponse{
        StatusCode: 200,
        Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
        Body:       []byte(`<html><body><h1>Hello World</h1></body></html>`),
    }, nil
}
```

## 常见问题

### Q: 如何处理跨域请求？

A: 在响应头中添加 CORS 头：

```go
return &plugin.RouterResponse{
    StatusCode: 200,
    Headers: map[string]string{
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*",
        "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE",
    },
    Body: data,
}, nil
```

### Q: 如何访问数据库？

A: 通过宿主提供的 API 接口（需要扩展支持），或使用插件内嵌数据库（如 BoltDB）。

### Q: 如何处理文件上传？

A: 解析 `multipart/form-data`：

```go
func handleUpload(req *http.Request) (*plugin.RouterResponse, error) {
    err := req.ParseMultipartForm(10 << 20) // 10MB
    if err != nil {
        return nil, err
    }
    
    file, _, err := req.FormFile("file")
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    // 处理文件内容
    // ...
    
    return &plugin.RouterResponse{
        StatusCode: 200,
        Body:       []byte("Upload successful"),
    }, nil
}
```

### Q: 如何实现后台任务？

A: 使用定时器实现：

```go
// 注册定时器（可取消）
tm := plugin.GetTimerManager()
timerID := tm.RegisterDelayTimer(ctx, 60000, func() {
    // 每分钟执行一次
    p.syncData()
})

// 需要取消时
tm.CancelTimer(ctx, timerID)
```

**注意**：插件环境中**不支持使用 goroutine**，请使用定时器来实现后台任务。

## 参考资源

- [示例插件代码](https://github.com/mimusic-org/mimusic-plugin-example)
- [插件协议定义](https://github.com/mimusic-org/plugin/tree/main/api/pbplugin/plugin.proto)
- [插件 API 文档](https://github.com/mimusic-org/plugin/blob/main/README.md)

---

**最后更新**: 2026-02-24  
**维护者**: MiMusic 团队
