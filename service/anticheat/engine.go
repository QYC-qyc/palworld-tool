package anticheat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/viper"
	"go.etcd.io/bbolt"
	"paladmin/internal/config"
	"paladmin/internal/database"
	"paladmin/internal/logger"
	"paladmin/internal/tool"
)

// Engine 反作弊引擎：持有规则、扫描器、处置器与冷却
type Engine struct {
	db         *bbolt.DB
	cfg        *config.AnticheatConfig
	executor   *ActionExecutor
	saveScan   *SaveScanner
	liveScan   *LiveScanner
	rules      map[string]Rule
	data       *GameData
	cooldown   map[string]time.Time
	mu         sync.Mutex
}

// New 构建引擎并加载规则
func New(db *bbolt.DB, cfg *config.AnticheatConfig) *Engine {
	data := LoadGameData(cfg.DataDir)
	e := &Engine{
		db:       db,
		cfg:      cfg,
		executor: NewActionExecutor(db, cfg),
		cooldown: map[string]time.Time{},
		data:     data,
	}
	e.rules = e.loadRules()
	e.saveScan = NewSaveScanner(e.rules, data)
	e.liveScan = NewLiveScanner(e.rules)
	return e
}

// ReloadData 重新加载游戏数据表（版本更新后调用）
func (e *Engine) ReloadData() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.data = ReloadGameData(e.cfg.DataDir)
	e.saveScan = NewSaveScanner(e.rules, e.data)
}

// loadRules 内置默认规则，再用 config 中的覆盖
func (e *Engine) loadRules() map[string]Rule {
	rules := map[string]Rule{}
	for _, r := range DefaultRules() {
		rules[r.ID] = r
	}
	for id, override := range e.cfg.Rules {
		r, ok := rules[id]
		if !ok {
			continue
		}
		if override.Enabled != nil {
			r.Enabled = *override.Enabled
		}
		if override.Severity != "" {
			r.Severity = Severity(override.Severity)
		}
		if len(override.Actions) > 0 {
			acts := make([]Action, 0, len(override.Actions))
			for _, a := range override.Actions {
				acts = append(acts, Action(a))
			}
			r.Actions = acts
		}
		if override.Reason != "" {
			r.Reason = override.Reason
		}
		if override.Params != nil {
			if r.Params == nil {
				r.Params = map[string]interface{}{}
			}
			for k, v := range override.Params {
				r.Params[k] = v
			}
		}
		rules[id] = r
	}
	return rules
}

// Rules 返回当前规则列表
func (e *Engine) Rules() []Rule {
	out := make([]Rule, 0, len(e.rules))
	for _, r := range e.rules {
		out = append(out, r)
	}
	return out
}

// UpdateRule 临时更新某条规则（运行期）
func (e *Engine) UpdateRule(id string, enabled *bool, severity string, actions []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	r, ok := e.rules[id]
	if !ok {
		return
	}
	if enabled != nil {
		r.Enabled = *enabled
	}
	if severity != "" {
		r.Severity = Severity(severity)
	}
	if len(actions) > 0 {
		acts := make([]Action, 0, len(actions))
		for _, a := range actions {
			acts = append(acts, Action(a))
		}
		r.Actions = acts
	}
	e.rules[id] = r
}

// ScanSave 执行存档扫描并处置
func (e *Engine) ScanSave(players []database.Player) int {
	if !e.cfg.Enabled {
		return 0
	}
	findings := e.saveScan.Scan(players)
	return e.process(findings)
}

// ScanLive 执行在线行为扫描
func (e *Engine) ScanLive(players []database.OnlinePlayer, whitelist []database.PlayerW) int {
	if !e.cfg.Enabled || !e.cfg.ScanLive {
		return 0
	}
	findings := e.liveScan.Scan(players, whitelist)
	return e.process(findings)
}

// ProcessExternal 处理外部（如 PalDefender）上报的 Finding
func (e *Engine) ProcessExternal(f Finding) {
	if !e.cfg.Enabled {
		return
	}
	f.Source = "paldefender"
	e.process([]Finding{f})
}

func (e *Engine) process(findings []Finding) int {
	count := 0
	for _, f := range findings {
		key := f.Rule.ID + "|" + f.PlayerUID + "|" + f.PalInstID
		e.mu.Lock()
		if t, ok := e.cooldown[key]; ok && time.Since(t) < time.Duration(e.cfg.Cooldown)*time.Second {
			e.mu.Unlock()
			continue
		}
		e.cooldown[key] = time.Now()
		e.mu.Unlock()

		// 保存证据
		if e.cfg.Evidence && len(f.Evidence) == 0 && len(f.Detail) > 0 {
			f.Evidence, _ = json.Marshal(f.Detail)
		}
		if e.cfg.Evidence && len(f.Evidence) > 0 {
			dir := e.cfg.EvidenceDir
			if dir == "" {
				dir = "./evidence"
			}
			_ = os.MkdirAll(dir, 0755)
		}

		alert := alertFromFinding(f)
		// 存证据路径
		if len(f.Evidence) > 0 {
			_ = os.MkdirAll(e.cfg.EvidenceDir, 0755)
			path := filepath.Join(e.cfg.EvidenceDir, alert.RuleID+"_"+f.PlayerUID+".json")
			if err := os.WriteFile(path, f.Evidence, 0644); err == nil {
				alert.EvidencePath = path
			}
		}
		saved, err := SaveAlert(e.db, alert)
		if err != nil {
			logger.Errorf("保存告警失败: %v", err)
			continue
		}
		logger.Warnf("[反作弊] %s 玩家=%s 规则=%s", f.Title, f.Nickname, f.Rule.ID)

		// 告警通知（Webhook，按严重度过滤）
		e.notify(f)

		// 处置
		if err := e.executor.Execute(f); err != nil {
			logger.Errorf("处置失败: %v", err)
		}
		_ = saved
		count++
	}
	return count
}

// notify 按配置通过 Webhook 推送告警
func (e *Engine) notify(f Finding) {
	webhookURL := viper.GetString("notify.webhook_url")
	if webhookURL == "" {
		return
	}
	minSeverity := viper.GetString("notify.min_severity")
	if !severityAtLeast(string(f.Rule.Severity), minSeverity) {
		return
	}
	text := fmt.Sprintf("**规则**: %s\n**玩家**: %s (%s)\n**详情**: %s",
		f.Rule.ID, f.Nickname, f.PlayerUID, f.Title)
	payload := tool.WebhookPayload{
		Title:    "[反作弊] " + f.Title,
		Text:     text,
		Severity: string(f.Rule.Severity),
		Fields:   f.Detail,
	}
	go func() {
		if err := tool.SendWebhook(viper.GetString("notify.webhook_type"), webhookURL, payload); err != nil {
			logger.Warnf("Webhook 推送失败: %v", err)
		}
	}()
}

// severityAtLeast 判断 s 是否达到阈值
func severityAtLeast(s, threshold string) bool {
	order := map[string]int{"info": 1, "warn": 2, "critical": 3, "": 1}
	return order[s] >= order[threshold]
}
