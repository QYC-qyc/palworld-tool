// 从参考项目前端数据生成合法帕鲁/物品/词条清单
const fs = require('fs');
const path = require('path');

const ref = path.join(__dirname, '..', '_reference', 'web', 'src', 'assets');
const out = path.join(__dirname, '..', 'data', 'gamedata');

const palJson = JSON.parse(fs.readFileSync(path.join(ref, 'pal.json'), 'utf8'));
const itemsJson = JSON.parse(fs.readFileSync(path.join(ref, 'items.json'), 'utf8'));
const skillJson = JSON.parse(fs.readFileSync(path.join(ref, 'skill.json'), 'utf8'));

// 帕鲁：取中文名映射下的合法 PalID（排除 BOSS_/GYM_ 前缀用于普通合法性校验）
const zhPals = palJson.zh || palJson.en || {};
const palIds = Object.keys(zhPals);
const legalPalIds = palIds.filter(id => !id.startsWith('BOSS_') && !id.startsWith('GYM_'));
const bossPalIds = palIds.filter(id => id.startsWith('BOSS_') || id.startsWith('GYM_'));

// 物品：items.json 结构可能是 {zh:{id:name}} 或 {id:{...}}
const itemSource = itemsJson.zh || itemsJson.en || itemsJson;
const legalItemIds = Object.keys(itemSource).filter(k => !k.startsWith('_'));

// 被动词条：skill.json
const skillSource = skillJson.zh || skillJson.en || skillJson;
const legalPassives = Object.keys(skillSource).filter(k =>
  k.toLowerCase().includes('passive') || k.startsWith('PAL_') || k.includes('up') || k.includes('Rare') || k.includes('Legend')
);

const data = {
  legal_pal_ids: legalPalIds.sort(),
  boss_pal_ids: bossPalIds.sort(),
  legal_item_ids: legalItemIds.sort(),
  legal_passives: [...new Set(legalPassives)].sort(),
  _counts: {
    legal_pal_ids: legalPalIds.length,
    boss_pal_ids: bossPalIds.length,
    legal_item_ids: legalItemIds.length,
    legal_passives: legalPassives.length,
  },
};

fs.writeFileSync(path.join(out, 'pal_ids.json'), JSON.stringify(
  { legal: legalPalIds.sort(), boss: bossPalIds.sort() }, null, 2));
fs.writeFileSync(path.join(out, 'item_ids.json'), JSON.stringify(legalItemIds.sort(), null, 2));
fs.writeFileSync(path.join(out, 'passive_ids.json'), JSON.stringify([...new Set(legalPassives)].sort(), null, 2));

console.log('generated:', data._counts);
