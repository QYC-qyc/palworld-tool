#!/bin/bash
# 自动清理旧 tag，只保留最近 N 个（本地 + Gitee origin + GitHub github）
# 用法: ./scripts/cleanup-tags.sh [保留数量，默认5]
#
# 注意：只删 tag，不删 GitHub/Gitee release（release 保留）。
# 需在仓库根目录执行，且已配置 origin(gitee) 和 github 两个 remote。

set -e

KEEP=${1:-5}

# 按版本号排序，取要删除的 tag
OLD_TAGS=$(git tag | sort -V | head -n -"$KEEP")

if [ -z "$OLD_TAGS" ]; then
  echo "没有需要清理的旧 tag（保留最近 $KEEP 个）"
  exit 0
fi

COUNT=$(echo "$OLD_TAGS" | wc -l)
echo "将删除 $COUNT 个旧 tag（保留最近 $KEEP 个）："
echo "$OLD_TAGS" | tr '\n' ' '
echo ""
echo ""

# 本地删除
echo ">> 删除本地 tag..."
echo "$OLD_TAGS" | while read -r t; do
  [ -n "$t" ] && git tag -d "$t"
done

# 远程删除参数
REFS=$(echo "$OLD_TAGS" | while read -r t; do
  [ -n "$t" ] && echo -n ":refs/tags/$t "
done)

echo ">> 删除 Gitee(origin) 远程 tag..."
eval "git push origin $REFS"

echo ">> 删除 GitHub(github) 远程 tag..."
eval "git -c http.sslVerify=false push github $REFS"

echo ""
echo "完成。剩余 tag："
git tag | sort -V | tr '\n' ' '
echo ""
