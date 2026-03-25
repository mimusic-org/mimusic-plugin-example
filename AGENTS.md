# MiMusic 插件开发规范

本文档详细说明了如何为 MiMusic 音乐服务器开发 WebAssembly 插件。

## 目录

- [快速开始](#快速开始)
- [插件架构](#插件架构)
- [项目结构](#项目结构)
- [开发步骤](#开发步骤)
- [核心 API 使用](#核心-api-使用)
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
