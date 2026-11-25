package permissionroutes

import "strings"

// Covers reports whether a stored policy path authorises a registered route
// template. Both use Gin route syntax. Unlike Casbin keyMatch2, a :param policy
// segment only covers a :param route segment: a detail grant must never match
// a static sibling such as /hosts/export, and an export grant must never match
// /hosts/:id. A trailing "*" policy segment keeps its explicit wildcard meaning.
func Covers(routePath, policyPath string) bool {
	routePath = strings.TrimSpace(routePath)
	policyPath = strings.TrimSpace(policyPath)
	if routePath == "" || policyPath == "" {
		return false
	}
	route := strings.Split(routePath, "/")
	policy := strings.Split(policyPath, "/")
	for i, segment := range policy {
		if segment == "*" && i == len(policy)-1 {
			return i <= len(route)
		}
		if i >= len(route) {
			return false
		}
		if isParamSegment(segment) {
			if !isParamSegment(route[i]) {
				return false
			}
			continue
		}
		if route[i] != segment {
			return false
		}
	}
	return len(route) == len(policy)
}

func isParamSegment(segment string) bool {
	return len(segment) > 1 && segment[0] == ':'
}
