package pkg

// NormalizeListLimit bounds provider list pages. Every k8s list handler shares
// the same 100-default/500-max contract, so the rule lives here instead of one
// copy per resource package.
func NormalizeListLimit(limit int64) int64 {
	if limit <= 0 {
		return 100
	}
	if limit > 500 {
		return 500
	}
	return limit
}
