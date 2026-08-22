// Package paldefender 封装 PalDefender 内置 REST API 的客户端。
// 所有调用由 PalAdmin 后端在本机回环代理，Token 不暴露给浏览器。
package paldefender

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"palworld-panel/internal/database"
	"palworld-panel/service"
)

// ErrNotConfigured 表示 PalDefender REST API 未启用或 Token 未配置。
var ErrNotConfigured = errors.New("paldefender: API 未配置（未启用或 Token 为空）")

// defaultBasePath PalDefender REST API 的默认前缀。
const defaultBasePath = "/v1/pdapi"

// Client 是 PalDefender REST API 的 HTTP 客户端。
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// Load 从面板动态设置中读取 PalDefender 配置并构建客户端。
// Token 为空即视为未配置（REST 的启停由 PalDefender 自身 RESTConfig.json 决定）。
func Load(db *database.Store) (*Client, error) {
	token := service.GetSetting(db, service.SettingPalDefenderToken)
	if token == "" {
		return nil, ErrNotConfigured
	}

	host := service.GetSetting(db, service.SettingPalDefenderHost)
	if host == "" {
		host = "127.0.0.1"
	}
	port := service.GetSetting(db, service.SettingPalDefenderPort)
	if port == "" {
		port = "17993"
	}
	basePath := service.GetSetting(db, service.SettingPalDefenderBasePath)
	if basePath == "" {
		basePath = defaultBasePath
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	basePath = strings.TrimSuffix(basePath, "/")

	return &Client{
		baseURL:    fmt.Sprintf("http://%s:%s%s", host, port, basePath),
		token:      token,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// pdError 对应 PalDefender 通用错误响应结构 {Error:{Code,Message}}。
type pdError struct {
	Err struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	} `json:"Error"`
}

func (e pdError) Error() string {
	if e.Err.Code != "" && e.Err.Message != "" {
		return fmt.Sprintf("PalDefender %s: %s", e.Err.Code, e.Err.Message)
	}
	if e.Err.Message != "" {
		return e.Err.Message
	}
	return "PalDefender 请求失败"
}

// do 发起一次已鉴权的 HTTP 请求，返回响应体原始字节。
// path 为 base path 之后的路径（需以 / 开头）；query 可选。
func (c *Client) do(method, path string, query url.Values, body any) ([]byte, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("paldefender: 序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("paldefender: 构建请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// 连接被拒绝通常是 REST API 未启用或游戏服/PalDefender 未运行
		errStr := err.Error()
		if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "connectex") {
			return nil, fmt.Errorf("无法连接 PalDefender REST API（%s）：请确认游戏服已启动、PalDefender 已加载，且 RESTConfig.json 中 Enabled=true 后重启过服务器", c.baseURL)
		}
		return nil, fmt.Errorf("paldefender: 请求失败: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("paldefender: 读取响应失败: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var pe pdError
		if err := json.Unmarshal(raw, &pe); err == nil && (pe.Err.Code != "" || pe.Err.Message != "") {
			return nil, pe
		}
		return nil, fmt.Errorf("paldefender: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	return raw, nil
}

// Version 返回版本信息（健康检查/连通性测试）。
func (c *Client) Version() (json.RawMessage, error) {
	return c.do(http.MethodGet, "/version", nil, nil)
}

// ListPlayers 返回所有已知玩家。
func (c *Client) ListPlayers() ([]byte, error) {
	return c.do(http.MethodGet, "/players", nil, nil)
}

// GetPlayer 返回单个玩家详情。
func (c *Client) GetPlayer(id string) ([]byte, error) {
	return c.do(http.MethodGet, "/player/"+url.PathEscape(id), nil, nil)
}

// Kick 踢出在线玩家，reason 可为空。
func (c *Client) Kick(id, reason string) ([]byte, error) {
	return c.do(http.MethodPost, "/kick/"+url.PathEscape(id), nil, map[string]string{"Reason": reason})
}

// Ban 封禁玩家，ip 为 true 时同时封禁 IP。
func (c *Client) Ban(id, reason string, ip bool) ([]byte, error) {
	return c.do(http.MethodPost, "/ban/"+url.PathEscape(id), nil, map[string]any{"Reason": reason, "IP": ip})
}

// Unban 解封用户（参数为 user_id，非 PlayerUID）。
func (c *Client) Unban(userID, reason string) ([]byte, error) {
	return c.do(http.MethodPost, "/unban/"+url.PathEscape(userID), nil, map[string]string{"Reason": reason})
}

// BanIP 封禁 IP。
func (c *Client) BanIP(ip, reason string) ([]byte, error) {
	return c.do(http.MethodPost, "/banip/"+url.PathEscape(ip), nil, map[string]string{"Reason": reason})
}

// UnbanIP 解封 IP。
func (c *Client) UnbanIP(ip, reason string) ([]byte, error) {
	return c.do(http.MethodPost, "/unbanip/"+url.PathEscape(ip), nil, map[string]string{"Reason": reason})
}

// Banlist 查询封禁列表，透传 query 参数。
func (c *Client) Banlist(query url.Values) ([]byte, error) {
	return c.do(http.MethodGet, "/banlist", query, nil)
}

// Broadcast 全服广播聊天消息。
func (c *Client) Broadcast(msg string) ([]byte, error) {
	return c.do(http.MethodPost, "/Broadcast", nil, map[string]string{"Message": msg})
}

// Alert 发送高优先级警报。
func (c *Client) Alert(msg string) ([]byte, error) {
	return c.do(http.MethodPost, "/Alert", nil, map[string]string{"Message": msg})
}

// SendPlayerMessage 向单个玩家发送私聊/公会/日志消息。
// sendType 为空时默认 PlayerChat。
func (c *Client) SendPlayerMessage(userID, message, sendType string) ([]byte, error) {
	if sendType == "" {
		sendType = "PlayerChat"
	}
	return c.do(http.MethodPost, "/SendPlayerMessage", nil, map[string]string{
		"SendType": sendType,
		"Message":  message,
		"UserID":   userID,
	})
}

// ListGuilds 返回公会摘要列表。
func (c *Client) ListGuilds() ([]byte, error) {
	return c.do(http.MethodGet, "/guilds", nil, nil)
}

// GetGuild 返回公会详情。
func (c *Client) GetGuild(id string) ([]byte, error) {
	return c.do(http.MethodGet, "/guild/"+url.PathEscape(id), nil, nil)
}

// DeleteBase 按营地 GUID 删除据点/营地（破坏性操作）。
func (c *Client) DeleteBase(baseCampID string) ([]byte, error) {
	return c.do(http.MethodPost, "/deletebase/"+url.PathEscape(baseCampID), nil, map[string]struct{}{})
}

// ReloadConfig 热重载 PalDefender 配置。
func (c *Client) ReloadConfig() ([]byte, error) {
	return c.do(http.MethodPost, "/ReloadConfig", nil, map[string]struct{}{})
}
