package sonarbaseline

func NormalizeLabel(label string) string {
	if label == "" {
		return "baseline"
	}
	return label
}
