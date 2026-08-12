// 从参考项目 skill.json 提取全部被动词条 ID
const fs = require('fs');
const path = require('path');

const skillJson = JSON.parse(fs.readFileSync(
  path.join(__dirname, '..', '_reference', 'web', 'src', 'assets', 'skill.json'), 'utf8'));

const source = skillJson.zh || skillJson.en || {};
const ids = Object.keys(source).filter(k => k && k.length < 80).sort();

fs.writeFileSync(
  path.join(__dirname, '..', 'data', 'gamedata', 'passive_ids.json'),
  JSON.stringify(ids, null, 2));

console.log('passive ids:', ids.length);
// 统计类别分布
const buckets = {};
for (const id of ids) {
  const key = id.split('_').slice(0, 2).join('_');
  buckets[key] = (buckets[key] || 0) + 1;
}
console.log(buckets);
