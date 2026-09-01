@echo off
REM Pantheon-Ops v1.2.0 真机验收测试脚本 (Windows)
REM 目标服务器: 115.190.251.143

echo ==========================================
echo Pantheon-Ops v1.2.0 真机验收测试
echo ==========================================
echo.

set SERVER_IP=115.190.251.143
set API_PORT=8080
set WEB_PORT=3000
set API_BASE_URL=http://%SERVER_IP%:%API_PORT%
set WEB_BASE_URL=http://%SERVER_IP%:%WEB_PORT%

echo 目标服务器: %SERVER_IP%
echo API 地址: %API_BASE_URL%
echo Web 地址: %WEB_BASE_URL%
echo.

REM 检查网络连接
echo ==========================================
echo 步骤 1: 检查网络连接
echo ==========================================
ping -n 1 %SERVER_IP% >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] 服务器可达
) else (
    echo [FAIL] 服务器不可达
    pause
    exit /b 1
)
echo.

REM 检查 API 服务
echo ==========================================
echo 步骤 2: 检查 API 服务
echo ==========================================
curl -s -o nul -w "%%{http_code}" %API_BASE_URL%/health >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] API 服务响应正常
) else (
    echo [WARN] API 服务可能未启动
)
echo.

REM 检查 Web 服务
echo ==========================================
echo 步骤 3: 检查 Web 服务
echo ==========================================
curl -s -o nul -w "%%{http_code}" %WEB_BASE_URL% >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] Web 服务响应正常
) else (
    echo [WARN] Web 服务可能未启动
)
echo.

REM 进入测试目录
cd /d "%~dp0..\e2e-tests"

REM 检查依赖
echo ==========================================
echo 步骤 4: 检查测试环境
echo ==========================================
if not exist "node_modules\" (
    echo 正在安装依赖...
    call npm install
)

REM 检查 Playwright
where npx >nul 2>&1
if %errorlevel% neq 0 (
    echo [FAIL] npx 未找到，请安装 Node.js
    pause
    exit /b 1
)

echo [OK] 测试环境就绪
echo.

REM 配置环境变量
echo ==========================================
echo 步骤 5: 配置测试环境变量
echo ==========================================
set BASE_URL=%API_BASE_URL%
set API_TOKEN=test-token
echo BASE_URL=%BASE_URL%
echo API_TOKEN=****** (已隐藏)
echo.

REM 提示用户
echo ==========================================
echo 准备运行冒烟测试
echo ==========================================
echo.
echo 测试将包含 30 个用例，预计耗时 20-30 秒
echo.
echo 按任意键开始测试，或按 Ctrl+C 取消...
pause >nul

REM 运行测试
echo.
echo ==========================================
echo 开始运行冒烟测试...
echo ==========================================
echo.

call npm test

set TEST_EXIT_CODE=%errorlevel%

echo.
echo ==========================================
echo 冒烟测试完成
echo ==========================================
echo.

if %TEST_EXIT_CODE% equ 0 (
    echo [SUCCESS] 冒烟测试通过
    echo.
    echo ==========================================
    echo 下一步: Web UI 验收
    echo ==========================================
    echo.
    echo 请在浏览器中访问以下地址进行 Web 验收：
    echo.
    echo 1. ArgoCD 管理: %WEB_BASE_URL%/argocd/servers
    echo 2. Velero 备份: %WEB_BASE_URL%/velero/backups
    echo 3. 监控告警: %WEB_BASE_URL%/observability/alerts
    echo 4. 文件分发: %WEB_BASE_URL%/filedist/files
    echo 5. 服务实例: %WEB_BASE_URL%/service-instances
    echo.
    echo 详细验收计划: .harness\v1.2-tasks\REAL_MACHINE_ACCEPTANCE_PLAN.md
    echo.

    REM 打开测试报告
    echo 按任意键查看测试报告...
    pause >nul
    call npx playwright show-report
) else (
    echo [FAIL] 冒烟测试失败
    echo.
    echo 查看详细报告：
    echo   npx playwright show-report
    echo.
    pause
    exit /b 1
)

echo.
echo 测试完成！
pause
