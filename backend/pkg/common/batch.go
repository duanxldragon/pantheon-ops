package common

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
