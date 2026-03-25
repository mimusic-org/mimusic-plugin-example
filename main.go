//go:build wasip1
// +build wasip1

// Package main 实现了 mimusic 系统的示例插件。
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/mimusic-org/plugin/api/pbplugin"
	"github.com/mimusic-org/plugin/api/plugin"
)

// main 函数是 Go 编译为 Wasm 所必需的。
func main() {}

// Plugin 实现了小米设备插件的功能。
type Plugin struct{}

// init 将 Plugin 实现注册到插件框架中。
func init() {
	plugin.RegisterPlugin(&Plugin{})
}

// GetPluginInfo 返回此插件的元数据。
func (m Plugin) GetPluginInfo(ctx context.Context, request *emptypb.Empty) (*pbplugin.GetPluginInfoResponse, error) {
	return &pbplugin.GetPluginInfoResponse{
		Success: true,
		Message: "成功获取插件信息",
		Info: &pbplugin.PluginInfo{
			Name:        "示例插件",
			Version:     "1.0.0",
			Description: "这是一个示例功能",
			Author:      "MiMusic Team",
			Homepage:    "https://github.com/mimusic-org/mimusic-plugin-example",
			EntryPath:   "/example",
		},
	}, nil
}

// Init 在宿主应用程序加载插件时初始化插件。
func (m *Plugin) Init(ctx context.Context, request *pbplugin.InitRequest) (*emptypb.Empty, error) {
	fmt.Println("正在初始化")

	rm := plugin.GetRouterManager()

	rm.RegisterRouter(ctx, "GET", "/exmaple/", func(req *http.Request) (*plugin.RouterResponse, error) {
		return &plugin.RouterResponse{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
			Body:       []byte(`<html><body><h1>示例插件</h1></body></html>`),
		}, nil
	})

	slog.Info("服务插件路由注册完成")
	return &emptypb.Empty{}, nil
}

// Deinit 在宿主应用程序卸载插件时清理资源。
func (m *Plugin) Deinit(ctx context.Context, request *emptypb.Empty) (*emptypb.Empty, error) {
	fmt.Println("正在反初始化示例插件")
	return &emptypb.Empty{}, nil
}
