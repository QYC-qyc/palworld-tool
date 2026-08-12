package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/viper"
)

// PalDefenderClient 对接 PalDefender 内置 REST API（默认 :17993）
type PalDefenderClient struct {
	baseURL string
	token   string
	timeout time.Duration
	client  *http.Client
}

// NewPalDefender 从配置构建客户端
func NewPalDefender() *PalDefenderClient {
	timeout := viper.GetInt("paldefender.timeout")
	if timeout <= 0 {
		timeout = 5
	}
	return &PalDefenderClient{
		baseURL: viper.GetString("paldefender.address"),
		token:   viper.GetString("paldefender.token"),
		timeout: time.Duration(timeout) * time.Second,
		client:  &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}
}

// Available 是否配置并可达
func (p *PalDefenderClient) Available() bool {
	return p.baseURL != "" && p.token != ""
}

func (p *PalDefenderClient) call(method, endpoint string, body interface{}) ([]byte, error) {
	full, err := url.JoinPath(p.baseURL, endpoint)
	if err != nil {
		return nil, err
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, full, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("paldefender %s %s: %d %s", method, endpoint, resp.StatusCode, string(data))
	}
	return data, nil
}

// ---------- 数据读取 ----------

// Version 版本与健康检查
func (p *PalDefenderClient) Version() (map[string]interface{}, error) {
	data, err := p.call("GET", "/v1/pdapi/version", nil)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	return m, json.Unmarshal(data, &m)
}

// Players 列出所有玩家
func (p *PalDefenderClient) Players() ([]map[string]interface{}, error) {
	data, err := p.call("GET", "/v1/pdapi/players", nil)
	if err != nil {
		return nil, err
	}
	var res map[string][]map[string]interface{}
	if err := json.Unmarshal(data, &res); err != nil {
		// 兼容直接返回数组
		var arr []map[string]interface{}
		if err2 := json.Unmarshal(data, &arr); err2 != nil {
			return nil, err
		}
		return arr, nil
	}
	return res["players"], nil
}

// Player 单个玩家详情
func (p *PalDefenderClient) Player(id string) (map[string]interface{}, error) {
	data, err := p.call("GET", "/v1/pdapi/player/"+id, nil)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	return m, json.Unmarshal(data, &m)
}

// Pals 玩家帕鲁
func (p *PalDefenderClient) Pals(id string) ([]map[string]interface{}, error) {
	data, err := p.call("GET", "/v1/pdapi/pals/"+id, nil)
	if err != nil {
		return nil, err
	}
	var arr []map[string]interface{}
	return arr, json.Unmarshal(data, &arr)
}

// Items 玩家物品
func (p *PalDefenderClient) Items(id string) ([]map[string]interface{}, error) {
	data, err := p.call("GET", "/v1/pdapi/items/"+id, nil)
	if err != nil {
		return nil, err
	}
	var arr []map[string]interface{}
	return arr, json.Unmarshal(data, &arr)
}

// Guilds 公会列表
func (p *PalDefenderClient) Guilds() ([]map[string]interface{}, error) {
	data, err := p.call("GET", "/v1/pdapi/guilds", nil)
	if err != nil {
		return nil, err
	}
	var arr []map[string]interface{}
	return arr, json.Unmarshal(data, &arr)
}

// Banlist 查询封禁列表（支持 query，如 ?active=true&entryType=user）
func (p *PalDefenderClient) Banlist(query string) ([]map[string]interface{}, error) {
	path := "/v1/pdapi/banlist"
	if query != "" {
		path += "?" + query
	}
	data, err := p.call("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var arr []map[string]interface{}
	return arr, json.Unmarshal(data, &arr)
}

// ---------- 处置 ----------

// Kick 踢出玩家（id 为 UserId/PlayerUID）
func (p *PalDefenderClient) Kick(id, reason string) error {
	body := map[string]string{}
	if reason != "" {
		body["Reason"] = reason
	}
	_, err := p.call("POST", "/v1/pdapi/kick/"+id, body)
	return err
}

// Ban 封禁用户，可选同时封 IP
func (p *PalDefenderClient) Ban(id, reason string, banIP bool) error {
	body := map[string]interface{}{"Reason": reason, "IP": banIP}
	_, err := p.call("POST", "/v1/pdapi/ban/"+id, body)
	return err
}

// Unban 解封用户
func (p *PalDefenderClient) Unban(id, reason string) error {
	body := map[string]string{"Reason": reason}
	_, err := p.call("POST", "/v1/pdapi/unban/"+id, body)
	return err
}

// BanIP 封禁 IP
func (p *PalDefenderClient) BanIP(ip, reason, userID string) error {
	body := map[string]string{"Reason": reason}
	if userID != "" {
		body["UserId"] = userID
	}
	_, err := p.call("POST", "/v1/pdapi/banip/"+ip, body)
	return err
}

// UnbanIP 解封 IP
func (p *PalDefenderClient) UnbanIP(ip, reason string) error {
	body := map[string]string{"Reason": reason}
	_, err := p.call("POST", "/v1/pdapi/unbanip/"+ip, body)
	return err
}

// ---------- 消息 ----------

// Broadcast 全服聊天广播
func (p *PalDefenderClient) Broadcast(message string) error {
	_, err := p.call("POST", "/v1/pdapi/Broadcast", map[string]string{"Message": message})
	return err
}

// Alert 发送醒目警报
func (p *PalDefenderClient) Alert(message string) error {
	_, err := p.call("POST", "/v1/pdapi/Alert", map[string]string{"Message": message})
	return err
}

// SendMessage 向玩家发消息
// sendType: PlayerChat / PlayerLogNormal / PlayerLogImportant / PlayerLogVeryImportant
func (p *PalDefenderClient) SendMessage(ids []string, sendType, message string) error {
	body := map[string]interface{}{
		"SendType": sendType,
		"Message":  message,
	}
	if len(ids) == 1 {
		body["UserID"] = ids[0]
	} else {
		body["UserIDs"] = ids
	}
	_, err := p.call("POST", "/v1/pdapi/SendPlayerMessage", body)
	return err
}

// DeleteBase 删除据点
func (p *PalDefenderClient) DeleteBase(baseCampID string) error {
	_, err := p.call("POST", "/v1/pdapi/deletebase/"+baseCampID, map[string]interface{}{})
	return err
}

// ReloadConfig 让 PalDefender 重载配置/封禁名单
func (p *PalDefenderClient) ReloadConfig() error {
	_, err := p.call("POST", "/v1/pdapi/ReloadConfig", map[string]interface{}{})
	return err
}
