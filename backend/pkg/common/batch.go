package common

// MaxBatchIDs 单次批量操作允许的最大 ID 数量，防止恶意超大批量请求拖垮服务。
const MaxBatchIDs = 1000

type BatchDeleteReq struct {
	IDs []uint64 `json:"ids" binding:"required"`
}

type BatchDeleteFailure struct {
	ID     uint64 `json:"id"`
	Reason string `json:"reason"`
}

type BatchDeleteResp struct {
	DeletedCount int                  `json:"deletedCount"`
	FailedCount  int                  `json:"failedCount"`
	Failures     []BatchDeleteFailure `json:"failures"`
}

func BatchDelete(ids []uint64, deleteOne func(uint64) error) BatchDeleteResp {
	normalized := NormalizeUint64IDs(ids)
	resp := BatchDeleteResp{
		Failures: []BatchDeleteFailure{},
	}
	// 超过上限时整体拒绝（不做部分执行），语义清晰且可被前端明确提示。
	if len(normalized) > MaxBatchIDs {
		for _, id := range normalized {
			resp.Failures = append(resp.Failures, BatchDeleteFailure{ID: id, Reason: "request.batch.too_large"})
		}
		resp.FailedCount = len(resp.Failures)
		return resp
	}
	for _, id := range normalized {
		if err := deleteOne(id); err != nil {
			resp.Failures = append(resp.Failures, BatchDeleteFailure{ID: id, Reason: ResolveErrorMessageKey(err, "request.failed")})
			continue
		}
		resp.DeletedCount++
	}
	resp.FailedCount = len(resp.Failures)
	return resp
}

func NormalizeUint64IDs(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	normalized := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized
}
