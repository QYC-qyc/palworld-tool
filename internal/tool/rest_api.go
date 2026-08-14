package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"paladmin/internal/database"
)

var httpClient = &http.Client{}

func callAPI(method, api string, param []byte) ([]byte, error) {
	addr := viper.GetString("rest.address")
	username := viper.GetString("rest.username")
	password := viper.GetString("rest.password")
	timeout := viper.GetInt("rest.timeout")
	return callAPIWith(method, api, param, addr, username, password, timeout)
}

// callAPIWith 用显式传入的连接参数发起 REST 请求（供测试连接使用）
func callAPIWith(method, api string, param []byte, addr, username, password string, timeout int) ([]byte, error) {
	fullURL, err := url.JoinPath(addr, api)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, fullURL, bytes.NewReader(param))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, password)

	if timeout <= 0 {
		timeout = 5
	}
	httpClient.Timeout = time.Duration(timeout) * time.Second
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("REST 返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return b, nil
}

// TestRest 用给定的地址和密码测试 REST API 连通性，返回服务器版本
func TestRest(addr, password string) (string, error) {
	resp, err := callAPIWith("GET", "/v1/api/info", nil, addr, "admin", password, 5)
	if err != nil {
		return "", err
	}
	var data ResponseInfo
	if err := json.Unmarshal(resp, &data); err != nil {
		return "", err
	}
	return data.Version, nil
}

type ResponseInfo struct {
	Version     string `json:"version"`
	ServerName  string `json:"servername"`
	Description string `json:"description"`
}

// Info 获取服务器基本信息
func Info() (map[string]string, error) {
	resp, err := callAPI("GET", "/v1/api/info", nil)
	if err != nil {
		return nil, err
	}
	var data ResponseInfo
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, err
	}
	return map[string]string{
		"version": data.Version,
		"name":    data.ServerName,
	}, nil
}

type ResponseMetrics struct {
	ServerFps        int     `json:"serverfps"`
	CurrentPlayerNum int     `json:"currentplayernum"`
	ServerFrameTime  float64 `json:"serverframetime"`
	MaxPlayerNum     int     `json:"maxplayernum"`
	Uptime           int     `json:"uptime"`
}

// Metrics 获取服务器性能指标
func Metrics() (map[string]interface{}, error) {
	resp, err := callAPI("GET", "/v1/api/metrics", nil)
	if err != nil {
		return nil, err
	}
	var data ResponseMetrics
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"server_fps":          data.ServerFps,
		"current_player_num":  data.CurrentPlayerNum,
		"server_frame_time":   data.ServerFrameTime,
		"max_player_num":      data.MaxPlayerNum,
		"uptime":              data.Uptime,
	}, nil
}

type ResponsePlayer struct {
	Name      string  `json:"name"`
	PlayerId  string  `json:"playerId"`
	UserId    string  `json:"userId"`
	Ip        string  `json:"ip"`
	Ping      float64 `json:"ping"`
	LocationX float64 `json:"location_x"`
	LocationY float64 `json:"location_y"`
	Level     int     `json:"level"`
}

type ResponsePlayers struct {
	Players []ResponsePlayer `json:"players"`
}

// ShowPlayers 获取在线玩家列表
func ShowPlayers() ([]database.OnlinePlayer, error) {
	resp, err := callAPI("GET", "/v1/api/players", nil)
	if err != nil {
		return nil, err
	}
	var data ResponsePlayers
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, err
	}
	online := make([]database.OnlinePlayer, 0, len(data.Players))
	for _, p := range data.Players {
		online = append(online, database.OnlinePlayer{
			PlayerUid:  getPlayerUid(p.PlayerId),
			SteamId:    getSteamId(p.UserId),
			Nickname:   p.Name,
			Ip:         p.Ip,
			Ping:       p.Ping,
			LocationX:  p.LocationX,
			LocationY:  p.LocationY,
			Level:      int32(p.Level),
			LastOnline: time.Now(),
		})
	}
	return online, nil
}

func getSteamId(userId string) string {
	if userId != "" && strings.HasPrefix(userId, "steam_") {
		return strings.TrimPrefix(userId, "steam_")
	}
	return ""
}

func getPlayerUid(playerId string) string {
	if len(playerId) < 8 {
		return ""
	}
	hexPart := playerId[:8]
	decimalValue, err := strconv.ParseUint(hexPart, 16, 32)
	if err != nil {
		return ""
	}
	return strconv.FormatUint(decimalValue, 10)
}

type RequestUserID struct {
	UserID string `json:"userid"`
}

// KickPlayer 踢出玩家，steamId 为纯数字，内部加 steam_ 前缀
func KickPlayer(steamId string) error {
	b, err := json.Marshal(RequestUserID{UserID: withPrefix(steamId)})
	if err != nil {
		return err
	}
	_, err = callAPI("POST", "/v1/api/kick", b)
	return err
}

// BanPlayer 封禁玩家
func BanPlayer(steamId string) error {
	b, err := json.Marshal(RequestUserID{UserID: withPrefix(steamId)})
	if err != nil {
		return err
	}
	_, err = callAPI("POST", "/v1/api/ban", b)
	return err
}

// UnBanPlayer 解封玩家
func UnBanPlayer(steamId string) error {
	b, err := json.Marshal(RequestUserID{UserID: withPrefix(steamId)})
	if err != nil {
		return err
	}
	_, err = callAPI("POST", "/v1/api/unban", b)
	return err
}

func withPrefix(id string) string {
	if strings.HasPrefix(id, "steam_") || strings.HasPrefix(id, "gdk_") {
		return id
	}
	return "steam_" + id
}

type RequestBroadcast struct {
	Message string `json:"message"`
}

// Broadcast 全服广播
func Broadcast(message string) error {
	b, err := json.Marshal(RequestBroadcast{Message: message})
	if err != nil {
		return err
	}
	_, err = callAPI("POST", "/v1/api/announce", b)
	return err
}

type RequestShutdown struct {
	Waittime int    `json:"waittime"`
	Message  string `json:"message"`
}

// Shutdown 平滑关服
func Shutdown(seconds int, message string) error {
	b, err := json.Marshal(RequestShutdown{Waittime: seconds, Message: message})
	if err != nil {
		return err
	}
	_, err = callAPI("POST", "/v1/api/shutdown", b)
	return err
}

// DoExit 立即停止服务器
func DoExit() error {
	_, err := callAPI("POST", "/v1/api/stop", nil)
	return err
}
