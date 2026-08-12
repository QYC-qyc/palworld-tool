package anticheat

import "paladmin/internal/config"

func configStub() *config.AnticheatConfig {
	cfg := &config.AnticheatConfig{
		Enabled:     true,
		ScanLive:    true,
		Cooldown:    0,
		Evidence:    false,
		DataDir:     "../../data/gamedata",
		EvidenceDir: "",
	}
	cfg.Punish.Warn = true
	cfg.Punish.Ban = false
	cfg.Punish.Kick = false
	return cfg
}
