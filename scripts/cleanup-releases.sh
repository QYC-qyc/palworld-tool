#!/bin/bash
# 清理旧的 GitHub releases 和 tags，只保留最近 N 个
# 用法: GITHUB_TOKEN=xxx ./scripts/cleanup-releases.sh [保留数量，默认5]
#
# 需要在 GitHub Settings > Developer settings > Personal access tokens 生成 token，
# 勾选 repo 权限。

set -e

KEEP=${1:-5}
REPO="QYC-qyc/palworld-tool"

if [ -z "$GITHUB_TOKEN" ]; then
  echo "错误: 请设置 GITHUB_TOKEN 环境变量"
  echo "  export GITHUB_TOKEN=ghp_xxxx"
  exit 1
fi

API="https://api.github.com/repos/$REPO"

echo "获取 release 列表（保留最近 $KEEP 个）..."
RELEASES=$(curl -s -H "Authorization: token $GITHUB_TOKEN" "$API/releases?per_page=100")

# 提取 release ID 和 tag name（按发布时间倒序）
COUNT=$(echo "$RELEASES" | grep -c '"id":' || true)
echo "找到 $COUNT 个 release"

echo "$RELEASES" | grep -E '"id":|"tag_name":|"published_at":' | head -30 | while read -r line; do
  echo "  $line"
done

# 获取要删除的 release（跳过前 KEEP*3 行，因为每个 release 有 id/tag_name/published_at）
echo ""
echo "将删除旧 release..."
echo "$RELEASES" | python3 -c "
import json, sys, os
keep = int(os.environ.get('KEEP', '5'))
data = json.load(sys.stdin)
to_delete = data[keep:]
for r in to_delete:
    print(f\"{r['id']} {r['tag_name']}\")
" KEEP=$KEEP | while read -r rid tag; do
  if [ -z "$rid" ] || [ "$rid" = "None" ]; then continue; fi
  echo "删除 release $tag (ID: $rid)..."
  curl -s -X DELETE -H "Authorization: token $GITHUB_TOKEN" "$API/releases/$rid"
done

echo ""
echo "删除旧 tags..."
# 获取所有 tag（按创建时间倒序）
TAGS=$(curl -s -H "Authorization: token $GITHUB_TOKEN" "$API/tags?per_page=100")
echo "$TAGS" | python3 -c "
import json, sys, os
keep = int(os.environ.get('KEEP', '5'))
data = json.load(sys.stdin)
to_delete = data[keep:]
for t in to_delete:
    print(t['name'])
" KEEP=$KEEP | while read -r tag; do
  if [ -z "$tag" ]; then continue; fi
  echo "删除 tag $tag..."
  curl -s -X DELETE -H "Authorization: token $GITHUB_TOKEN" "$API/git/refs/tags/$tag"
done

echo ""
echo "完成！保留最近 $KEEP 个 release"
