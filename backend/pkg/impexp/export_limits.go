package impexp

// maxExportRows 统一同步导出的行数 hard cap。所有 CSV 导出必须以
// CapExportRows（或领域等价常量）下推 SQL LIMIT，禁止无界全表导出。
const maxExportRows = 10000

// MaxExportRows exposes the shared synchronous-export row cap for callers
// and tests.
func MaxExportRows() int { return maxExportRows }
