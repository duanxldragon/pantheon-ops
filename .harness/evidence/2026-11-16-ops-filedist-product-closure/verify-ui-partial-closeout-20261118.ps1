[CmdletBinding()]
param([ValidatePattern('^[0-9]{2}$')][string]$RunSuffix = '01', [switch]$SkipAppTypecheck)
$ErrorActionPreference = 'Stop'
$opsRoot = 'D:\workspace\go\pantheon-platform\pantheon-ops'
$evidenceRoot = Join-Path $opsRoot '.harness\evidence\2026-11-16-ops-filedist-product-closure'
$sourcePaths = @('api.ts','createTaskPayload.ts','taskState.ts','useTaskDetail.ts','useFileChoices.ts','useFiledistListQuery.ts','FiledistList.tsx','FiledistDetail.tsx','filedist.css') | ForEach-Object { 'frontend/src/modules/business/filedist/' + $_ }
function Get-OwnedSnapshot {
    return @($sourcePaths | ForEach-Object { @{ path = $_; sha256 = (Get-FileHash -LiteralPath (Join-Path $opsRoot $_)).Hash.ToLowerInvariant() } })
}
$snapshotBefore = Get-OwnedSnapshot
$commands = @()
$checks = @(
    @{ name = 'eslint'; cwd = (Join-Path $opsRoot 'frontend'); args = @('node_modules/eslint/bin/eslint.js','src/modules/business/filedist','--max-warnings','0') },
    @{ name = 'regressions'; cwd = $opsRoot; args = @('.harness/evidence/2026-11-16-ops-filedist-product-closure/run-ui-lifetime-regression-20261118.mjs') }
)
if (!$SkipAppTypecheck) { $checks += @{ name = 'app-typecheck'; cwd = (Join-Path $opsRoot 'frontend'); args = @('node_modules/typescript/bin/tsc','-b','--pretty','false') } }
foreach ($check in $checks) {
    $logPath = Join-Path $evidenceRoot ('ui-final-' + $check.name + '-20261118-' + $RunSuffix + '.log')
    if (Test-Path -LiteralPath $logPath) { throw 'Preserve old evidence; choose a fresh final log name.' }
    $startedAt = (Get-Date).ToUniversalTime().ToString('o')
    Push-Location $check.cwd
    try { & node @($check.args) *> $logPath; $checkExit = $LASTEXITCODE } finally { Pop-Location }
    $commands += @{ command = 'node ' + ($check.args -join ' '); cwd = $check.cwd; startedAt = $startedAt; finishedAt = (Get-Date).ToUniversalTime().ToString('o'); exitCode = $checkExit; log = [IO.Path]::GetRelativePath($opsRoot,$logPath).Replace('\','/') }
}
Push-Location $opsRoot
try {
    $diffLog = Join-Path $evidenceRoot ('ui-final-diff-20261118-' + $RunSuffix + '.log')
    if (Test-Path -LiteralPath $diffLog) { throw 'Preserve old diff evidence.' }
    $startedAt = (Get-Date).ToUniversalTime().ToString('o')
    & git diff --check -- frontend/src/modules/business/filedist *> $diffLog
    $commands += @{ command = 'git diff --check -- frontend/src/modules/business/filedist'; cwd = $opsRoot; startedAt = $startedAt; finishedAt = (Get-Date).ToUniversalTime().ToString('o'); exitCode = $LASTEXITCODE; log = [IO.Path]::GetRelativePath($opsRoot,$diffLog).Replace('\','/') }
} finally { Pop-Location }
$snapshotAfter = Get-OwnedSnapshot
$unchanged = ($snapshotBefore | ConvertTo-Json -Depth 4 -Compress) -eq ($snapshotAfter | ConvertTo-Json -Depth 4 -Compress)
$receipt = @{ commands = $commands; sourceBefore = $snapshotBefore; sourceAfter = $snapshotAfter; sourceUnchanged = $unchanged; scope = 'Nine FileDist UI files; no assertion of full-tree stability'; appTypecheckDeferredToRoot = [bool]$SkipAppTypecheck; verifiedAtUTC = (Get-Date).ToUniversalTime().ToString('o') }
$receiptPath = Join-Path $evidenceRoot ('ui-final-verification-20261118-' + $RunSuffix + '.json')
if (Test-Path -LiteralPath $receiptPath) { throw 'Preserve old verification receipt.' }
$receipt | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $receiptPath -Encoding utf8
$commands | Select-Object command,exitCode | Format-Table -AutoSize
Write-Output "Owned source unchanged: $unchanged"
if (!$unchanged -or @($commands | Where-Object exitCode -ne 0).Count) { exit 1 }
exit 0
