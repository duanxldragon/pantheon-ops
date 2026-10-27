#!/bin/bash

# Pantheon-Ops v1.2.0 真机验收测试脚本
# 目标服务器: 115.190.251.143

set -e

echo "=========================================="
echo "Pantheon-Ops v1.2.0 真机验收测试"
echo "=========================================="
echo ""

# 配置
SERVER_IP="115.190.251.143"
API_PORT="8080"
WEB_PORT="3000"
API_BASE_URL="http://${SERVER_IP}:${API_PORT}"
WEB_BASE_URL="http://${SERVER_IP}:${WEB_PORT}"

echo "目标服务器: ${SERVER_IP}"
echo "API 地址: ${API_BASE_URL}"
echo "Web 地址: ${WEB_BASE_URL}"
echo ""

# 步骤 1: 检查网络连接
echo "步骤 1: 检查网络连接..."
if ping -n 1 ${SERVER_IP} > /dev/null 2>&1; then
    echo "✓ 服务器可达"
else
    echo "✗ 服务器不可达"
    exit 1
fi
echo ""

# 步骤 2: 检查 API 服务
echo "步骤 2: 检查 API 服务..."
if curl -s -o /dev/null -w "%{http_code}" ${API_BASE_URL}/health | grep -q "200"; then
    echo "✓ API 服务正常"
else
    echo "⚠ API 服务可能未启动或健康检查端点不存在"
fi
echo ""

# 步骤 3: 检查 Web 服务
echo "步骤 3: 检查 Web 服务..."
if curl -s -o /dev/null -w "%{http_code}" ${WEB_BASE_URL} | grep -q "200"; then
    echo "✓ Web 服务正常"
else
    echo "⚠ Web 服务可能未启动"
fi
echo ""

# 步骤 4: 准备测试环境
echo "步骤 4: 准备测试环境..."
cd e2e-tests

if [ ! -d "node_modules" ]; then
    echo "安装依赖..."
    npm install
fi

if ! npx playwright --version > /dev/null 2>&1; then
    echo "安装 Playwright..."
    npx playwright install chromium
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
echo "=========================================="
echo "开始运行冒烟测试..."
echo "=========================================="
echo ""

npm test

TEST_EXIT_CODE=$?

echo ""
echo "=========================================="
echo "冒烟测试完成"
echo "=========================================="
echo ""

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✓ 冒烟测试通过"
    echo ""
    echo "=========================================="
    echo "请继续进行 Web UI 验收"
    echo "=========================================="
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
    echo "npx playwright show-report"
    echo ""
    exit 1
fi
