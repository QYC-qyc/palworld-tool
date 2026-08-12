package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// WebhookPayload 通用告警负载
type WebhookPayload struct {
	Title    string                 `json:"title"`
	Text     string                 `json:"text"`
	Severity string                 `json:"severity"`
	Fields   map[string]interface{} `json:"fields,omitempty"`
}

var webhookClient = &http.Client{Timeout: 8 * time.Second}

// SendWebhook 根据类型发送到企业微信/钉钉/Discord/通用 webhook
func SendWebhook(webhookType, url string, payload WebhookPayload) error {
	if url == "" {
		return nil
	}
	var body []byte
	var contentType = "application/json"
	var err error

	switch strings.ToLower(webhookType) {
	case "dingtalk":
		body, err = json.Marshal(map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"title": payload.Title,
				"text":  fmt.Sprintf("### %s\n\n%s", payload.Title, payload.Text),
			},
		})
	case "wechat", "wecom":
		body, err = json.Marshal(map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"content": fmt.Sprintf("### %s\n%s", payload.Title, payload.Text),
			},
		})
	case "discord":
		var fields []map[string]string
		for k, v := range payload.Fields {
			fields = append(fields, map[string]string{"name": k, "value": fmt.Sprint(v)})
		}
		color := 0x808080
		switch payload.Severity {
		case "critical":
			color = 0xE53935
		case "warn":
			color = 0xFB8C00
		case "info":
			color = 0x4CAF50
		}
		body, err = json.Marshal(map[string]interface{}{
			"embeds": []map[string]interface{}{{
				"title":  payload.Title,
				"description": payload.Text,
				"color":  color,
				"fields": fields,
				"footer": map[string]string{"text": "PalAdmin 反作弊"},
			}},
		})
	default: // generic
		body, err = json.Marshal(payload)
	}
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := webhookClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook 返回状态码 %d", resp.StatusCode)
	}
	return nil
}
