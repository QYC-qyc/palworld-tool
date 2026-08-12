package anticheat

import (
	"testing"

	"paladmin/internal/database"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	cfg := configStub()
	e := New(nil, cfg)
	return e
}

func TestIllegalPalStats(t *testing.T) {
	e := testEngine(t)
	players := []database.Player{{
		TersePlayer: database.TersePlayer{PlayerUid: "1", Nickname: "Cheater"},
		Pals: []*database.Pal{
			{InstanceID: "p1", Type: "Anubis", Level: 200, MaxHp: 999999, TalentHP: 999},
		},
	}}
	findings := e.saveScan.Scan(players)
	hit := map[string]bool{}
	for _, f := range findings {
		hit[f.Rule.ID] = true
	}
	if !hit["S001"] {
		t.Errorf("期望命中 S001(帕鲁属性越界), 实际命中: %v", hit)
	}
	if !hit["S002"] {
		t.Errorf("期望命中 S002(天赋越界), 实际命中: %v", hit)
	}
}

func TestBossPal(t *testing.T) {
	e := testEngine(t)
	players := []database.Player{{
		TersePlayer: database.TersePlayer{PlayerUid: "2"},
		Pals: []*database.Pal{
			{InstanceID: "b1", Type: "BOSS_Anubis", IsBoss: true, Level: 50, TalentHP: 50},
		},
	}}
	findings := e.saveScan.Scan(players)
	found := false
	for _, f := range findings {
		if f.Rule.ID == "S004" {
			found = true
		}
	}
	if !found {
		t.Errorf("期望命中 S004(Boss帕鲁), 实际: %d 条", len(findings))
	}
}

func TestDuplicatePal(t *testing.T) {
	e := testEngine(t)
	mk := func(uid, inst string) database.Player {
		return database.Player{
			TersePlayer: database.TersePlayer{PlayerUid: uid},
			Pals: []*database.Pal{
				{InstanceID: inst, Type: "Anubis", Level: 50, TalentHP: 50, Rank: 4},
			},
		}
	}
	// 同一 InstanceID 出现在两个玩家身上 -> S007
	findings := e.saveScan.Scan([]database.Player{mk("1", "DUP-1"), mk("2", "DUP-1")})
	found := false
	for _, f := range findings {
		if f.Rule.ID == "S007" {
			found = true
		}
	}
	if !found {
		t.Errorf("期望命中 S007(复制帕鲁), 实际: %d 条", len(findings))
	}
}

func TestPlayerLevelOverflow(t *testing.T) {
	e := testEngine(t)
	players := []database.Player{{
		TersePlayer: database.TersePlayer{PlayerUid: "3", Level: 999, MaxHp: 100},
	}}
	findings := e.saveScan.Scan(players)
	found := false
	for _, f := range findings {
		if f.Rule.ID == "S008" {
			found = true
		}
	}
	if !found {
		t.Errorf("期望命中 S008(玩家等级越界), 实际: %d 条", len(findings))
	}
}

func TestItemStackOverflow(t *testing.T) {
	e := testEngine(t)
	players := []database.Player{{
		TersePlayer: database.TersePlayer{PlayerUid: "4"},
		Items: &database.Items{
			CommonContainerId: []*database.Item{{ItemId: "arrow", StackCount: 999999}},
		},
	}}
	findings := e.saveScan.Scan(players)
	found := false
	for _, f := range findings {
		if f.Rule.ID == "S005" {
			found = true
		}
	}
	if !found {
		t.Errorf("期望命中 S005(物品堆叠越界), 实际: %d 条", len(findings))
	}
}
