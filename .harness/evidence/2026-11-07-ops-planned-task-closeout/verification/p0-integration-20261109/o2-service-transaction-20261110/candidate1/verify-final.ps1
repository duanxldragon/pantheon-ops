$ErrorActionPreference = 'Stop'
$repo = 'D:\workspace\go\pantheon-platform\pantheon-ops'
$evidence = Join-Path $repo '.harness\evidence\2026-11-07-ops-planned-task-closeout\verification\p0-integration-20261109\o2-service-transaction-20261110'
$oldEvidence = Join-Path $repo '.harness\evidence\2026-11-07-ops-planned-task-closeout\verification\p0-integration-20261109\o2-engine-ownership-20261110'
$env:GOROOT = 'D:\cache\go\pkg\mod\golang.org\toolchain@v0.0.1-go1.26.10.windows-amd64'
$env:PATH = $env:GOROOT + '\bin;D:\msys64\mingw64\bin;' + $env:PATH
$env:CGO_ENABLED = '1'
$env:GOTOOLCHAIN = 'local'
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:GOCACHE = Join-Path $repo '.local\o2-service-transaction-gocache'
$env:PANTHEON_TEST_DSN = 'root@tcp(127.0.0.1:33316)/ops_p0_acceptance?parseTime=true&loc=Local'
$env:PANTHEON_TEST_DSN_REQUIRED = '1'
function Get-ScopedHashes {
 @('capability','service','deploy') | ForEach-Object { Get-ChildItem -LiteralPath (Join-Path $repo ('backend\modules\business\'+$_)) -Recurse -File } |
 Sort-Object FullName | ForEach-Object { [pscustomobject]@{path=$_.FullName.Substring($repo.Length+1).Replace('\','/');sha256=(Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()} }
}
$before = @(Get-ScopedHashes)
$before | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $evidence 'final-source-before.json')
$started = [DateTime]::UtcNow.ToString('o')
Set-Location (Join-Path $repo 'backend')
# Reuse the existing helper strictly without its optional cleanup arguments.
& go run (Join-Path $oldEvidence 'inspect-owned-fixtures.go') 1> (Join-Path $evidence 'fixture-inventory-before.json') 2> (Join-Path $evidence 'fixture-inventory-before.stderr.txt')
$inventoryBeforeExit = $LASTEXITCODE
$matrix = @(
 @{name='race';args=@('test','-mod=readonly','-race','-count=1','-timeout=25m','./modules/business/capability/...','./modules/business/service/...','./modules/business/deploy/...','-v')},
 @{name='vet';args=@('vet','-mod=readonly','./modules/business/capability/...','./modules/business/service/...','./modules/business/deploy/...')},
 @{name='build';args=@('build','-mod=readonly','./modules/business/capability/...','./modules/business/service/...','./modules/business/deploy/...')},
 @{name='test-inventory';args=@('test','-mod=readonly','-list','.', './modules/business/capability/...','./modules/business/service/...','./modules/business/deploy/...')}
)
$results = @()
foreach ($check in $matrix) {
 $arguments = $check.args
 & go @arguments *> (Join-Path $evidence ('final-'+$check.name+'.txt'))
 $code = $LASTEXITCODE
 $results += [pscustomobject]@{name=$check.name;command=('go '+($arguments -join ' '));exitCode=$code}
 Write-Output ($check.name+' exit '+$code)
}
$initial = @(Get-Content -Raw -LiteralPath (Join-Path $evidence 'source-before.json') | ConvertFrom-Json)
$changed = @($before | Where-Object { $entry=$_; -not ($initial | Where-Object { $_.path -ceq $entry.path -and $_.sha256 -ceq $entry.sha256 }) })
$goChanged = @($changed | Where-Object path -like '*.go' | ForEach-Object { $_.path.Substring('backend/'.Length) })
& gopls check @goChanged *> (Join-Path $evidence 'final-native-diagnostics.txt')
$results += [pscustomobject]@{name='native-diagnostics';command=('gopls check '+($goChanged -join ' '));exitCode=$LASTEXITCODE}
& git diff --check -- modules/business/capability modules/business/service modules/business/deploy *> (Join-Path $evidence 'final-diff-check.txt')
$results += [pscustomobject]@{name='diff-check';command='git diff --check -- modules/business/capability modules/business/service modules/business/deploy';exitCode=$LASTEXITCODE}
& go run (Join-Path $oldEvidence 'inspect-owned-fixtures.go') 1> (Join-Path $evidence 'fixture-inventory-after.json') 2> (Join-Path $evidence 'fixture-inventory-after.stderr.txt')
$inventoryAfterExit = $LASTEXITCODE
$after = @(Get-ScopedHashes)
$after | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $evidence 'final-source-after.json')
$stable = ($before | ConvertTo-Json -Compress) -ceq ($after | ConvertTo-Json -Compress)
$delta = @($changed | ForEach-Object { $entry=$_; $original=$initial | Where-Object path -CEQ $entry.path; [pscustomobject]@{path=$entry.path;oldSHA256=$original.sha256;newSHA256=$entry.sha256} })
$delta | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $evidence 'source-delta-final.json')
$raceLines = @(Get-Content -LiteralPath (Join-Path $evidence 'final-race.txt'))
$passes = @($raceLines | Where-Object { $_ -cmatch '^--- PASS: ([^ /]+) ' } | ForEach-Object { $Matches[1] })
$skips = @($raceLines | Where-Object { $_ -cmatch '^--- SKIP: ([^ /]+) ' } | ForEach-Object { $Matches[1] })
$fails = @($raceLines | Where-Object { $_ -cmatch '^--- FAIL: ([^ /]+) ' } | ForEach-Object { $Matches[1] })
$oldInventory = @(Get-Content -LiteralPath (Join-Path $oldEvidence 'stable-v2-race.txt') | Where-Object { $_ -cmatch '^--- (PASS|SKIP): ([^ /]+) ' } | ForEach-Object { $Matches[2] })
$missingPrior = @($oldInventory | Where-Object { $_ -cnotin $passes -and $_ -cnotin $skips })
$listed = @((Get-Content -LiteralPath (Join-Path $evidence 'final-test-inventory.txt')) | Where-Object { $_ -cmatch '^Test[A-Za-z0-9_]+$' })
$missingListed = @($listed | Where-Object { $_ -cnotin $passes -and $_ -cnotin $skips -and $_ -cnotin $fails })
$testFacts = [pscustomobject]@{passes=$passes;skips=$skips;failures=$fails;prior46SourceTopLevelTests=$oldInventory.Count;missingPriorTests=$missingPrior;listedTopLevelTests=$listed.Count;missingListedTests=$missingListed;independent47SourceRegressions=@('TestIndependentO2CancelBeforeFirstClaimSettles','TestIndependentO2ServiceProjectionRollsBackWithFinish');originalIndependentRegressionSHA256=(Get-FileHash -LiteralPath (Join-Path $oldEvidence 'continue-o2-final-independent-code-regression_test.go')).Hash.ToLowerInvariant()}
$testFacts | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $evidence 'test-facts-final.json')
$receipt = [pscustomobject]@{parent='2026-11-07-ops-planned-task-closeout';scope='backend/modules/business/{capability,service,deploy}/**';startedUTC=$started;finishedUTC=[DateTime]::UtcNow.ToString('o');sourceStable=$stable;fileCount=$after.Count;changedFileCount=$changed.Count;CGO_ENABLED=$env:CGO_ENABLED;MinGW='D:\msys64\mingw64\bin';toolchain=$env:GOROOT;GOCACHE=$env:GOCACHE;requiredMySQL='1';results=$results;topLevelPasses=$passes.Count;topLevelSkips=$skips;topLevelFailures=$fails;missingPriorTests=$missingPrior;missingListedTests=$missingListed;fixtureInventoryBeforeExit=$inventoryBeforeExit;fixtureInventoryAfterExit=$inventoryAfterExit;fullOpsRace='Dispatcher later';independentReview='required current-source CODE/ARCH re-review';currentSourceVMAcceptance='pending re-review and Dispatcher';P0Closure='not claimed'}
$receipt | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $evidence 'focused-final-receipt.json')
Write-Output ('source stable='+$stable+' top passes='+$passes.Count+' skips='+$skips.Count+' failures='+$fails.Count+' prior missing='+$missingPrior.Count+' inventory missing='+$missingListed.Count)
if (-not $stable -or @($results | Where-Object exitCode -ne 0).Count -gt 0 -or $missingPrior.Count -gt 0 -or $missingListed.Count -gt 0 -or $inventoryBeforeExit -ne 0 -or $inventoryAfterExit -ne 0) { exit 1 }
