#!/bin/bash

# 发布版本脚本
# 功能：更新版本号、提交更改、推送代码、创建或更新tag
# 使用方法: ./scripts/release.sh [版本号] [发布说明]
# 示例: ./scripts/release.sh v0.5.0 "新功能发布"

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# 检查参数
if [ $# -lt 1 ]; then
    echo -e "${RED}错误: 请提供版本号${NC}"
    echo "使用方法: $0 <版本号> [发布说明]"
    echo "示例: $0 v0.5.0 \"新功能发布\""
    exit 1
fi

VERSION=$1
RELEASE_MESSAGE=${2:-"Release $VERSION"}

# 验证版本号格式（vX.Y.Z）
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}错误: 版本号格式不正确，应为 vX.Y.Z (例如: v0.5.0)${NC}"
    exit 1
fi

# 移除 'v' 前缀用于比较
VERSION_NUM=${VERSION#v}

echo -e "${GREEN}开始发布版本: $VERSION${NC}"
echo ""

# 检查工作目录是否干净
if [ -n "$(git status --porcelain)" ]; then
    echo -e "${YELLOW}警告: 工作目录有未提交的更改${NC}"
    echo "当前更改："
    git status --short
    echo ""
    read -p "是否继续？(y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "已取消"
        exit 1
    fi
fi

# 检查是否在 main 分支
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo -e "${YELLOW}警告: 当前不在 main 分支，当前分支: $CURRENT_BRANCH${NC}"
    read -p "是否继续？(y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "已取消"
        exit 1
    fi
fi

# 更新版本号
echo -e "${GREEN}步骤 1/5: 更新版本号...${NC}"
VERSION_FILE="$PROJECT_ROOT/backend/cmd/server/main.go"

if [ ! -f "$VERSION_FILE" ]; then
    echo -e "${RED}错误: 未找到版本文件: $VERSION_FILE${NC}"
    exit 1
fi

# 备份文件
cp "$VERSION_FILE" "$VERSION_FILE.bak"

# 更新版本号
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' "s/Version   = \"v[0-9]\+\.[0-9]\+\.[0-9]\+\"/Version   = \"$VERSION\"/" "$VERSION_FILE"
else
    # Linux
    sed -i "s/Version   = \"v[0-9]\\+\\.[0-9]\\+\\.[0-9]\\+\"/Version   = \"$VERSION\"/" "$VERSION_FILE"
fi

# 验证更新
if ! grep -q "Version   = \"$VERSION\"" "$VERSION_FILE"; then
    echo -e "${RED}错误: 版本号更新失败${NC}"
    mv "$VERSION_FILE.bak" "$VERSION_FILE"
    exit 1
fi

rm "$VERSION_FILE.bak"
echo -e "${GREEN}✓ 版本号已更新为 $VERSION${NC}"
echo ""

# 提交更改
echo -e "${GREEN}步骤 2/5: 提交更改...${NC}"
git add "$VERSION_FILE"
git commit -m "chore: 更新版本号为 $VERSION" || {
    echo -e "${YELLOW}警告: 提交失败，可能没有更改${NC}"
}
echo -e "${GREEN}✓ 更改已提交${NC}"
echo ""

# 推送代码
echo -e "${GREEN}步骤 3/5: 推送代码到远程仓库...${NC}"
git push origin "$CURRENT_BRANCH" || {
    echo -e "${RED}错误: 推送失败${NC}"
    exit 1
}
echo -e "${GREEN}✓ 代码已推送${NC}"
echo ""

# 检查 tag 是否已存在
echo -e "${GREEN}步骤 4/5: 创建或更新 tag...${NC}"
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo -e "${YELLOW}Tag $VERSION 已存在，是否更新？(y/N)${NC}"
    read -p "" -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # 删除本地和远程 tag
        git tag -d "$VERSION"
        git push origin ":refs/tags/$VERSION" || true
        echo -e "${GREEN}✓ 已删除旧 tag${NC}"
    else
        echo "跳过 tag 更新"
        exit 0
    fi
fi

# 创建 tag
git tag -a "$VERSION" -m "$RELEASE_MESSAGE"
echo -e "${GREEN}✓ Tag $VERSION 已创建${NC}"
echo ""

# 推送 tag
echo -e "${GREEN}步骤 5/5: 推送 tag 到远程仓库...${NC}"
git push origin "$VERSION" || {
    echo -e "${RED}错误: Tag 推送失败${NC}"
    exit 1
}
echo -e "${GREEN}✓ Tag 已推送${NC}"
echo ""

# 显示发布信息
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}版本发布成功！${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo "版本号: $VERSION"
echo "发布说明: $RELEASE_MESSAGE"
echo "Tag: $VERSION"
echo "分支: $CURRENT_BRANCH"
echo ""
echo "下一步："
echo "  1. 运行编译脚本构建发布版本: ./scripts/build.sh $VERSION"
echo "  2. 在 GitHub 上创建 Release: https://github.com/funnywwh/prjflow/releases/new"
echo ""

