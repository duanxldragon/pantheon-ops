Independent source-only security review. Do not use tools or edit files. Review only the parameter projection below. Report proved P1/P2 leaks or correctness issues; give a minimal fix, or APPROVE for the supplied excerpt only. Max 150 words. Input snapshot values have been JSON decoded. Known secret set is complete unless complete=false. Do not infer release approval.
var taskPreviewIdentifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.@+-]*$`)
var taskPreviewPath = regexp.MustCompile(`^/[A-Za-z0-9_./-]+$`)
var taskPreviewRetention = regexp.MustCompile(`^[0-9]+(ms|s|m|h|d|w|y|B|KB|MB|GB|TB)$`)

func taskPreviewPublicText(value string, secrets map[string]string, complete bool) string {
if !complete || len(value) > taskPreviewValueLimit || strings.Contains(value, secretSentinelPrefix) || strings.IndexFunc(value, unicode.IsControl) >= 0 || redactDeploySecrets(value, secrets) != value {
return ""
}
return value
}

// Minimal-complexity: only the existing fixed renderer's public structural
// fields are displayable. Freeform params need a future explicit public schema;
// checking only a secret-looking key would allow nested or mislabeled secrets.
func taskPreviewParameterValue(key string, value any, secrets map[string]string, complete bool) string {
if isDeployTaskSecretParam(key) {
return ""
}
switch value.(type) {
case string, float64, bool:
default:
return ""
}
text := taskPreviewPublicText(anyToString(value), secrets, complete)
if text == "" {
return ""
}
switch key {
case "port", "apiPort", "consolePort", "httpPort", "webPort":
port, err := strconv.Atoi(text)
if err == nil && port > 0 && port <= 65535 {
return text
}
case "installRoot", "dataRoot":
if taskPreviewPath.MatchString(text) {
return text
}
case "serviceName", "hostname", "version":
if taskPreviewIdentifier.MatchString(text) {
return text
}
case "storageRetentionPolicy":
if taskPreviewRetention.MatchString(text) {
return text
}
case "action":
if validTaskAction(text) {
return text
}
}
return ""
}
