#!/bin/bash

# Pantheon-Ops v1.2.0 真机验收测试脚本
# 目标服务器: 由调用方通过 API_BASE_URL / WEB_BASE_URL 显式指定

set -e

# 配置
# 目标地址必须由调用方显式提供（含协议），脚本不内置任何默认端点：
# 隐式的明文 HTTP 默认值会把验收凭据发往错误的宿主。
API_BASE_URL="${API_BASE_URL:-}"
WEB_BASE_URL="${WEB_BASE_URL:-}"
SEPARATOR="=========================================="

if [[ -z "${API_BASE_URL}" || -z "${WEB_BASE_URL}" ]]; then
    echo "必须显式设置 API_BASE_URL 与 WEB_BASE_URL（含协议），例如 API_BASE_URL=https://ops.example.com:8080" >&2
    exit 1
fi

# 从 API 地址中取出主机名，用于连通性探测
API_HOST="${API_BASE_URL#*://}"
API_HOST="${API_HOST%%/*}"
API_HOST="${API_HOST%%:*}"

echo "${SEPARATOR}"
echo "Pantheon-Ops v1.2.0 真机验收测试"
echo "${SEPARATOR}"
echo ""
echo "API 地址: ${API_BASE_URL}"
echo "Web 地址: ${WEB_BASE_URL}"
echo ""

# 步骤 1: 检查网络连接
echo "步骤 1: 检查网络连接..."
if ping -n 1 "${API_HOST}" > /dev/null 2>&1; then
    echo "✓ 服务器可达"
else
    echo "✗ 服务器不可达"
    exit 1
fi
echo ""

# 步骤 2: 检查 API 服务
echo "步骤 2: 检查 API 服务..."
if curl -s -o /dev/null -w "%{http_code}" "${API_BASE_URL}/health" | grep -q "200"; then
    echo "✓ API 服务正常"
else
    echo "⚠ API 服务可能未启动或健康检查端点不存在"
fi
echo ""

# 步骤 3: 检查 Web 服务
echo "步骤 3: 检查 Web 服务..."
if curl -s -o /dev/null -w "%{http_code}" "${WEB_BASE_URL}" | grep -q "200"; then
    echo "✓ Web 服务正常"
else
    echo "⚠ Web 服务可能未启动"
fi
echo ""

# 步骤 4: 准备测试环境
echo "步骤 4: 准备测试环境..."
cd e2e-tests

PLAYWRIGHT_BIN="./node_modules/.bin/playwright"

if [[ ! -d "node_modules" ]]; then
    echo "安装依赖..."
    npm ci --ignore-scripts
fi

if ! "${PLAYWRIGHT_BIN}" --version > /dev/null 2>&1; then
    echo "安装 Playwright..."
    "${PLAYWRIGHT_BIN}" install chromium
fi

echo "✓ 测试环境就绪"
echo ""

# 步骤 5: 配置环境变量
echo "步骤 5: 配置测试环境变量..."
export BASE_URL="${API_BASE_URL}"
export API_TOKEN="${API_TOKEN:-test-token}"
echo "BASE_URL=${BASE_URL}"
echo "API_TOKEN=****** (已隐藏)"
echo ""

# 步骤 6: 运行冒烟测试
echo "${SEPARATOR}"
echo "开始运行冒烟测试..."
echo "${SEPARATOR}"
echo ""

TEST_EXIT_CODE=0
npm test || TEST_EXIT_CODE=$?

echo ""
echo "${SEPARATOR}"
echo "冒烟测试完成"
echo "${SEPARATOR}"
echo ""

if [[ ${TEST_EXIT_CODE} -eq 0 ]]; then
    echo "✓ 冒烟测试通过"
    echo ""
    echo "${SEPARATOR}"
    echo "请继续进行 Web UI 验收"
    echo "${SEPARATOR}"
    echo ""
    echo "访问以下地址进行 Web 验收："
    echo ""
    echo "1. ArgoCD 管理: ${WEB_BASE_URL}/argocd/servers"
    echo "2. Velero 备份: ${WEB_BASE_URL}/velero/backups"
    echo "3. 监控告警: ${WEB_BASE_URL}/observability/alerts"
    echo "4. 文件分发: ${WEB_BASE_URL}/filedist/files"
    echo "5. 服务实例: ${WEB_BASE_URL}/service-instances"
    echo ""
    echo "详细验收计划请参考: .harness/v1.2-tasks/REAL_MACHINE_ACCEPTANCE_PLAN.md"
    echo ""
else
    echo "✗ 冒烟测试失败"
    echo ""
    echo "查看详细报告："
    echo "${PLAYWRIGHT_BIN} show-report"
    echo ""
    exit 1
fi
