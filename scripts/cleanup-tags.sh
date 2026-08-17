#!/bin/bash
# 自动清理旧 tag，只保留最近 N 个（本地 + Gitee + GitHub）
# 用法: ./scripts/cleanup-tags.sh [保留数量，默认5]
#
# 注意：只删 tag，不删 release（release 保留）。
# Gitee→GitHub 自动同步只同步新增/更新、不同步删除，因此删除 tag 必须两端都删。
# 新 tag/分支推送只需推 Gitee（origin），会自动同步到 GitHub。

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

# Gitee 删除（自动同步会删除 GitHub 上的分支/提交，但 tag 删除不会同步，所以下面还要单独删 GitHub）
echo ">> 删除 Gitee(origin) 远程 tag..."
eval "git push origin $REFS"

# GitHub 单独删除（Gitee 自动同步不传播删除操作）
echo ">> 删除 GitHub(github) 远程 tag..."
eval "git -c http.sslVerify=false push github $REFS"

echo ""
echo "完成。剩余 tag："
git tag | sort -V | tr '\n' ' '
echo ""
