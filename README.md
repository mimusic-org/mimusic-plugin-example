# MiMusic 插件示例工程

这是一个 MiMusic WebAssembly 插件的最小示例，展示了插件的基本结构和生命周期实现。

## 项目结构

```
mimusic-plugin-example/
├── main.go      # 插件入口，实现生命周期方法和路由注册
├── Makefile     # 构建脚本
├── go.mod       # Go 模块定义
└── go.sum       # 依赖锁定
```

## 快速开始

### 环境要求

- Go 1.24+（支持 WASI 的工具链）
- Make（可选）

### 构建

```bash
# 使用 Makefile 构建
make build

# 或手动构建（必须添加 -buildmode=c-shared）
GOOS=wasip1 GOARCH=wasm go build -o example.wasm -buildmode=c-shared .
```

### 部署

将生成的 `example.wasm` 上传到 MiMusic 管理界面即可。

## 开发新插件

参考 [AGENTS.md](./AGENTS.md) 了解完整的插件开发规范，包括：

- 插件架构与生命周期
- 路由注册与 HTTP 处理器
- 静态资源管理
- 定时器使用
- 代码规范与最佳实践

## 相关资源

- [插件开发规范](./AGENTS.md)
- [示例插件代码](https://github.com/mimusic-org/mimusic-plugin-example)
- [插件协议定义](https://github.com/mimusic-org/plugin/tree/main/api/pbplugin/plugin.proto)
- [插件 API 文档](https://github.com/mimusic-org/plugin/blob/main/README.md)

## License

[Apache License 2.0](./LICENSE)
