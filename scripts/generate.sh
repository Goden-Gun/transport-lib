#!/bin/bash

# Protobuf 代码生成脚本
# 用于将 proto 文件生成为 Go 代码

set -e

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}开始生成 Protobuf 代码...${NC}"

# 检查 protoc 是否安装
if ! command -v protoc &> /dev/null; then
    echo -e "${RED}错误: protoc 未安装${NC}"
    echo "请访问 https://grpc.io/docs/protoc-installation/ 安装 protoc"
    exit 1
fi

# 检查 protoc-gen-go 是否安装
if ! command -v protoc-gen-go &> /dev/null; then
    echo -e "${YELLOW}protoc-gen-go 未安装，正在安装...${NC}"
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# 检查 protoc-gen-go-grpc 是否安装
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo -e "${YELLOW}protoc-gen-go-grpc 未安装，正在安装...${NC}"
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# 设置路径
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
PROTO_DIR="${ROOT_DIR}/proto"
GEN_DIR="${ROOT_DIR}/gen/go"

# 将 Go bin 目录添加到 PATH
export PATH="$PATH:$(go env GOPATH)/bin"

# 清理旧的生成文件
echo -e "${YELLOW}清理旧的生成文件...${NC}"
rm -rf "${GEN_DIR}"
mkdir -p "${GEN_DIR}"

# 生成 Go 代码
echo -e "${GREEN}生成 Go 代码...${NC}"
protoc \
    --proto_path="${PROTO_DIR}" \
    --go_out="${GEN_DIR}" \
    --go_opt=module=gga-transport-lib/gen/go \
    --go-grpc_out="${GEN_DIR}" \
    --go-grpc_opt=module=gga-transport-lib/gen/go \
    "${PROTO_DIR}/bridge/v1/bridge.proto"

echo -e "${GREEN}✓ Protobuf 代码生成完成！${NC}"
echo -e "${GREEN}生成文件位置: ${GEN_DIR}${NC}"

# 显示生成的文件
echo -e "${YELLOW}生成的文件：${NC}"
find "${GEN_DIR}" -type f -name "*.go" | sed "s|${GEN_DIR}/||"
