package impexp

var GovernanceExportHeaders = []string{
	"governanceScope",
	"governanceTags",
	"governanceProblemCount",
	"governanceBlockedBy",
	"governanceActions",
	"governanceScopeLabel",
	"governanceTagsLabel",
	"governanceBlockedByLabel",
	"governanceActionsLabel",
}

func CountGovernanceProblems(tags []string, problemTags map[string]struct{}) int {
	count := 0
	for _, tag := range tags {
		if _, ok := problemTags[tag]; ok {
			count++
		}
	}
	return count
}

var governanceScopeLabels = map[string]string{
	"dept": "Department",
	"post": "Post",
}

var governanceTagLabels = map[string]string{
	"clean":      "Healthy",
	"leaderless": "Leader Missing",
	"no-post":    "No Posts",
	"empty":      "Empty Department",
	"in-use":     "Assigned Members",
	"disabled":   "Disabled",
	"root":       "Root Department",
}

var governanceBlockedByLabels = map[string]string{
	"none":     "No Blocker",
	"children": "Child Departments",
	"posts":    "Posts",
	"users":    "Users",
}

var governanceActionLabels = map[string]string{
	"keep-observing":          "Keep Observing",
	"assign-leader":           "Assign Leader",
	"create-post":             "Create Post",
	"review-merge-or-delete":  "Review Merge or Delete",
	"clear-child-depts":       "Clear Child Departments",
	"clear-posts":             "Clear Posts",
	"clear-users":             "Clear Users",
	"reassign-users":          "Reassign Users",
	"review-status":           "Review Status",
	"delete-or-keep-disabled": "Delete or Keep Disabled",
}

func GovernanceScopeLabel(scope string) string {
	return governanceScopeLabels[scope]
}

func GovernanceValueLabels(values []string, dictionary map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	labels := make([]string, 0, len(values))
	for _, value := range values {
		if label, ok := dictionary[value]; ok && label != "" {
			labels = append(labels, label)
			continue
		}
		labels = append(labels, value)
	}
	return joinGovernanceLabels(labels)
}

func GovernanceTagLabels(values []string) string {
	return GovernanceValueLabels(values, governanceTagLabels)
}

func GovernanceBlockedByLabels(values []string) string {
	return GovernanceValueLabels(values, governanceBlockedByLabels)
}

func GovernanceActionLabels(values []string) string {
	return GovernanceValueLabels(values, governanceActionLabels)
}

func joinGovernanceLabels(values []string) string {
	if len(values) == 0 {
		return ""
	}
	result := values[0]
	for i := 1; i < len(values); i++ {
		result += " | " + values[i]
	}
	return result
}
