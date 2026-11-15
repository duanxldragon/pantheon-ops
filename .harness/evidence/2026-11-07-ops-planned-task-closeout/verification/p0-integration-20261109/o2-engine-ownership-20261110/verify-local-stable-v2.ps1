$ErrorActionPreference = 'Stop'
$repo = 'D:\workspace\go\pantheon-platform\pantheon-ops'
$evidence = Join-Path $repo '.harness\evidence\2026-11-07-ops-planned-task-closeout\verification\p0-integration-20261109\o2-engine-ownership-20261110'
$env:GOROOT = 'D:\cache\go\pkg\mod\golang.org\toolchain@v0.0.1-go1.26.10.windows-amd64'
$env:PATH = $env:GOROOT + '\bin;D:\msys64\mingw64\bin;' + $env:PATH
$env:CGO_ENABLED = '1'
$env:GOTOOLCHAIN = 'local'
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:GOCACHE = Join-Path $repo '.local\o2-engine-gocache'
$env:PANTHEON_TEST_DSN = 'root@tcp(127.0.0.1:33316)/ops_p0_acceptance?parseTime=true&loc=Local'
$env:PANTHEON_TEST_DSN_REQUIRED = '1'
function Get-SourceHashes {
 Get-ChildItem -LiteralPath (Join-Path $repo 'backend\modules\business\deploy') -Recurse -File |
 Sort-Object FullName |
 ForEach-Object { [pscustomobject]@{path=$_.FullName.Substring($repo.Length+1).Replace('\','/');sha256=(Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()} }
}
$before = @(Get-SourceHashes)
$before | ConvertTo-Json -Depth 4 | Set-Content (Join-Path $evidence 'stable-v2-source-before.json')
$started = [DateTime]::UtcNow.ToString('o')
Set-Location (Join-Path $repo 'backend')
$results = @()
$matrix = @(
 @{name='race';args=@('test','-mod=readonly','-race','-count=1','-timeout=25m','./modules/business/deploy/...','-v')},
 @{name='vet';args=@('vet','-mod=readonly','./modules/business/deploy/...')},
 @{name='build';args=@('build','-mod=readonly','./modules/business/deploy/...')}
)
foreach ($check in $matrix) {
 $arguments=$check.args
 & go @arguments *> (Join-Path $evidence ('stable-v2-'+$check.name+'.txt'))
 $code=$LASTEXITCODE
 $results += [pscustomobject]@{name=$check.name;command=('go '+($arguments -join ' '));exitCode=$code}
 Write-Output ($check.name+' exit '+$code)
}
$changed=@('attempt_ownership.go','attempt_ownership_test.go','deploy_service.go','deploy_reliability_test.go','deploy_ssh_transport_test.go','deploy_rollback.go','resume_scope_revocation_test.go','task_command.go','task_preview.go','target_selection.go') |
 ForEach-Object { 'modules/business/deploy/'+$_ }
& gopls check @changed *> (Join-Path $evidence 'stable-v2-native-diagnostics.txt')
$results += [pscustomobject]@{name='native-diagnostics';command=('gopls check '+($changed -join ' '));exitCode=$LASTEXITCODE}
& git diff --check -- modules/business/deploy *> (Join-Path $evidence 'stable-v2-diff-check.txt')
$results += [pscustomobject]@{name='diff-check';command='git diff --check -- modules/business/deploy';exitCode=$LASTEXITCODE}
$after=@(Get-SourceHashes)
$after | ConvertTo-Json -Depth 4 | Set-Content (Join-Path $evidence 'stable-v2-source-after.json')
$stable=($before | ConvertTo-Json -Compress) -ceq ($after | ConvertTo-Json -Compress)
$receipt=[pscustomobject]@{
 parent='2026-11-07-ops-planned-task-closeout';scope='backend/modules/business/deploy/**';
 startedUTC=$started;finishedUTC=[DateTime]::UtcNow.ToString('o');sourceStable=$stable;fileCount=$after.Count;
 cgo=$env:CGO_ENABLED;toolchain=$env:GOROOT;cache=$env:GOCACHE;requiredMySQL=$env:PANTHEON_TEST_DSN_REQUIRED;
 results=$results;fullTreeGate='Dispatcher/root final gate; not executed by this Generator';
 independentReview='pending implementation CODE/ARCH; no self-approval';acceptance='O2/P0 remains open'
}
$receipt | ConvertTo-Json -Depth 6 | Set-Content (Join-Path $evidence 'stable-v2-focused-receipt.json')
$receipt | ConvertTo-Json -Depth 6
if (-not $stable -or @($results | Where-Object exitCode -ne 0).Count -gt 0) { exit 1 }

