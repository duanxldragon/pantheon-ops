$ErrorActionPreference='Stop'
$opsRoot=(Resolve-Path -LiteralPath "$PSScriptRoot/../../../..").Path
$runtimeDir="$opsRoot/.tmp/ops-priority-runtime"
$webOwner=Get-Content -Raw "$runtimeDir/step-history-web-owner.json" | ConvertFrom-Json
$serviceOwner=Get-Content -Raw "$runtimeDir/step-history-services-owner.json" | ConvertFrom-Json
$race=Get-Content -Raw "$PSScriptRoot/o2-step-history-full-race-final-result.json" | ConvertFrom-Json
if ($race.exitCode -ne 0 -or -not $race.sourceHashesStable) { throw 'Final hash-stable race has not passed' }
$checks=@()
foreach ($binding in @(@{port=24485;exe=$webOwner.backendExecutable;hash=$webOwner.backendHash;launcher=$webOwner.backendPid},@{port=33316;exe=$serviceOwner.mysqlExecutable;hash=$serviceOwner.mysqlHash;launcher=$serviceOwner.mysqlLauncherPid},@{port=26316;exe=$serviceOwner.redisExecutable;hash=$serviceOwner.redisHash;launcher=$serviceOwner.redisLauncherPid})) {
 $listeners=@(Get-NetTCPConnection -State Listen -LocalPort $binding.port -ErrorAction SilentlyContinue)
 if ($listeners.Count -ne 1) { throw "Unexpected listener count on $($binding.port)" }
 $proc=Get-CimInstance Win32_Process -Filter "ProcessId=$($listeners[0].OwningProcess)"
 $expectedExe=(Resolve-Path -LiteralPath $binding.exe).Path
 if ($proc.ExecutablePath -ne $expectedExe -or (Get-FileHash -LiteralPath $expectedExe -Algorithm SHA256).Hash -ne $binding.hash) { throw "Executable/hash mismatch on $($binding.port)" }
 if ($proc.ProcessId -ne $binding.launcher -and $proc.ParentProcessId -ne $binding.launcher) { throw "Launcher lineage mismatch on $($binding.port)" }
 $checks+=@{port=$binding.port;pid=$proc.ProcessId;parent=$proc.ParentProcessId;executable=$expectedExe;hashVerified=$true}
}
$mysqlData=(Resolve-Path -LiteralPath $serviceOwner.mysqlData).Path
$redisData=(Resolve-Path -LiteralPath $serviceOwner.redisData).Path
$workspacePrefix=(Resolve-Path -LiteralPath $opsRoot).Path.TrimEnd('\')+'\'
foreach ($dataPath in @($mysqlData,$redisData)) { if (-not $dataPath.StartsWith($workspacePrefix,[StringComparison]::OrdinalIgnoreCase)) { throw 'Data directory is outside intended Ops workspace' } }
$mysqlProc=Get-CimInstance Win32_Process -Filter "ProcessId=$($checks[1].pid)"
$redisProc=Get-CimInstance Win32_Process -Filter "ProcessId=$($checks[2].pid)"
if (-not $mysqlProc.CommandLine.Replace('/','\').Contains($mysqlData) -or -not $redisProc.CommandLine.Replace('/','\').Contains($redisData)) { throw 'Exact service data directory argument mismatch' }
$serverData=& D:/MySQL/mysql/bin/mysql.exe --host=127.0.0.1 --port=33316 --user=root --batch --raw --skip-column-names --execute='SELECT @@datadir,@@GLOBAL.innodb_flush_log_at_trx_commit,@@GLOBAL.sync_binlog;'
if ($LASTEXITCODE -ne 0) { throw 'Server data verification failed' }
$fields=$serverData.Split("`t")
if ($fields.Count -ne 3 -or $fields[0].TrimEnd('\','/') -ne $mysqlData -or $fields[1] -ne '1' -or $fields[2] -ne '1') { throw 'Server datadir/durability mismatch' }
if (Get-NetTCPConnection -State Listen -LocalPort 5173 -ErrorAction SilentlyContinue) { throw 'Suite web listener remains; do not stop an unverified process' }
$checks | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath "$PSScriptRoot/o2-step-history-owned-processes.json" -Encoding utf8
Stop-Process -Id $webOwner.backendPid -ErrorAction Stop
& D:/Redis/redis-cli.exe -h 127.0.0.1 -p 26316 shutdown
if ($LASTEXITCODE -ne 0) { throw 'Owned Redis graceful shutdown failed' }
& D:/MySQL/mysql/bin/mysqladmin.exe --host=127.0.0.1 --port=33316 --user=root shutdown
if ($LASTEXITCODE -ne 0) { throw 'Owned MySQL graceful shutdown failed' }
Start-Sleep -Seconds 2
$remainingPorts=@(Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue | Where-Object { $_.LocalPort -in @(24485,33316,26316,5173,5175) } | Select-Object -ExpandProperty LocalPort)
$checkPids=@($checks | ForEach-Object { $_.pid }) + @($serviceOwner.mysqlLauncherPid,$serviceOwner.redisLauncherPid)
$remainingPids=@(Get-CimInstance Win32_Process | Where-Object { $_.ProcessId -in $checkPids } | Select-Object -ExpandProperty ProcessId)
$receipt=@{ownedProcesses=$checks;serverDatadir=$mysqlData;redisDirectory=$redisData;durability='1/1; unchanged';backendStop='verified own Stop-Process';webStop='verified own Vite stopped before suite; suite web stopped itself';mysqlRedisStop='graceful CLI shutdown';remainingPorts=$remainingPorts;remainingPids=$remainingPids;dataRetained=$true;otherServices='untouched';completedUtc=[DateTime]::UtcNow.ToString('o')}
$receipt | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath "$PSScriptRoot/o2-step-history-owned-cleanup.json" -Encoding utf8
$receipt | ConvertTo-Json -Depth 6
if ($remainingPids.Count -or $remainingPorts.Count) { throw 'Owned resources remain active; record second observation before any further stop' }