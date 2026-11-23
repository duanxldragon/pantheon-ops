package deploy

import (
	"strings"
	"testing"

	"gorm.io/datatypes"
)

// Fixed-template parameters are interpolated into shell commands, systemd unit
// files and YAML. validateFixedTemplateValues is the single shared safety gate
// for create/preview and render, so an unsafe value must be rejected before it
// can ever reach an executable script.
func TestFixedTemplateRejectsShellSubstitution(t *testing.T) {
	params := map[string]any{
		"action":      TaskActionInstall,
		"installRoot": "/data/nginx",
		"serviceName": "nginx$(touch /tmp/omx_fixed_injected)",
	}
	if err := validateFixedTemplateParams(TemplateCodeNginxSystemd, params); err == nil {
		t.Fatal("create path must reject serviceName shell substitution")
	}
	pkg := DeployPackage{
		Name:           "nginx",
		Version:        "1.25.3",
		ExecutionMode:  ExecutionModeFixed,
		TemplateCode:   TemplateCodeNginxSystemd,
		SourceFileName: "nginx-1.25.3.tar.gz",
		SourceURL:      "https://files.internal.local/nginx-1.25.3.tar.gz",
	}
	task := DeployTask{
		Action:         TaskActionInstall,
		TemplateParams: datatypes.JSON([]byte(`{"installRoot":"/data/nginx","serviceName":"nginx$(touch /tmp/omx_fixed_injected)"}`)),
	}
	if _, err := renderFixedTemplateScript(pkg, task); err == nil {
		t.Fatal("render must reject serviceName shell substitution for persisted records")
	}
}

func TestFixedTemplateRejectsQuoteAndNewline(t *testing.T) {
	params := map[string]any{
		"action":      TaskActionInstall,
		"installRoot": "/data/nginx",
		"serviceName": "nginx\"; touch /tmp/omx_fixed_quote; #",
	}
	if err := validateFixedTemplateParams(TemplateCodeNginxSystemd, params); err == nil {
		t.Fatal("create path must reject quoted serviceName injection")
	}
	params["serviceName"] = "nginx\nshutdown -h now"
	if err := validateFixedTemplateParams(TemplateCodeNginxSystemd, params); err == nil {
		t.Fatal("create path must reject newline in serviceName")
	}
}

func TestFixedTemplateRejectsPathTraversal(t *testing.T) {
	params := map[string]any{
		"action":      TaskActionInstall,
		"installRoot": "/data/nginx/../../etc",
		"serviceName": "nginx",
	}
	if err := validateFixedTemplateParams(TemplateCodeNginxSystemd, params); err == nil {
		t.Fatal("create path must reject path traversal installRoot")
	}
	params["installRoot"] = "/data/../bad"
	if err := validateFixedTemplateParams(TemplateCodeNginxSystemd, params); err == nil {
		t.Fatal("create path must reject .. segment in installRoot")
	}
}

func TestFixedTemplateRejectsDestructiveUninstallRoot(t *testing.T) {
	params := map[string]any{
		"action":      TaskActionUninstall,
		"installRoot": "/",
		"serviceName": "nginx",
	}
	if err := validateFixedTemplateParams(TemplateCodeNginxSystemd, params); err == nil {
		t.Fatal("uninstall must reject root / installRoot")
	}
	params["installRoot"] = "/etc"
	if err := validateFixedTemplateParams(TemplateCodeNginxSystemd, params); err == nil {
		t.Fatal("uninstall must reject /etc installRoot")
	}
	params["installRoot"] = "/data/nginx"
	if err := validateFixedTemplateParams(TemplateCodeNginxSystemd, params); err != nil {
		t.Fatalf("uninstall of /data/nginx must be allowed, got %v", err)
	}
}

func TestFixedTemplateRejectsInvalidPorts(t *testing.T) {
	for _, port := range []string{"0", "-1", "65536", "80abc", "12 34"} {
		params := map[string]any{
			"action":       TaskActionInstall,
			"installRoot":  "/data/mysql",
			"dataRoot":     "/data/mysql/data",
			"serviceName":  "mysqld",
			"port":         port,
			"rootPassword": "Secret_123",
		}
		if err := validateFixedTemplateParams(TemplateCodeMySQLSystemd, params); err == nil {
			t.Fatalf("port %q must be rejected", port)
		}
	}
	params := map[string]any{
		"action":       TaskActionInstall,
		"installRoot":  "/data/mysql",
		"dataRoot":     "/data/mysql/data",
		"serviceName":  "mysqld",
		"port":         "3306",
		"rootPassword": "Secret_123",
	}
	if err := validateFixedTemplateParams(TemplateCodeMySQLSystemd, params); err != nil {
		t.Fatalf("valid port 3306 must pass, got %v", err)
	}
}

func TestFixedTemplateAllowsShellSafePasswords(t *testing.T) {
	params := map[string]any{
		"action":       TaskActionInstall,
		"installRoot":  "/data/mysql",
		"dataRoot":     "/data/mysql/data",
		"serviceName":  "mysqld",
		"port":         "3306",
		"rootPassword": `P@ssw0rd-$!&*+#~`,
	}
	if err := validateFixedTemplateParams(TemplateCodeMySQLSystemd, params); err != nil {
		t.Fatalf("opaque shell-quoted password must be allowed, got %v", err)
	}
}

func TestFixedTemplateValidDefaultsRenderNormally(t *testing.T) {
	pkg := DeployPackage{
		Name:           "nginx",
		Version:        "1.25.3",
		ExecutionMode:  ExecutionModeFixed,
		TemplateCode:   TemplateCodeNginxSystemd,
		SourceFileName: "nginx-1.25.3.tar.gz",
		SourceURL:      "https://files.internal.local/nginx-1.25.3.tar.gz",
	}
	task := DeployTask{
		Action:         TaskActionInstall,
		TemplateParams: datatypes.JSON([]byte(`{"installRoot":"/data/nginx","serviceName":"nginx"}`)),
	}
	script, err := renderFixedTemplateScript(pkg, task)
	if err != nil {
		t.Fatalf("valid nginx fixed template must render, got %v", err)
	}
	if !strings.Contains(script, "SERVICE_NAME=\"nginx\"") {
		t.Fatalf("expected nginx serviceName in rendered script, got: %s", script[:120])
	}
}

// Package version is interpolated into MySQL/Redis shell literals even though
// it is package metadata, so an older package row must not bypass the gate.
func TestFixedTemplateRejectsUnsafePackageVersion(t *testing.T) {
	mysqlTask := DeployTask{
		Action:         TaskActionInstall,
		TemplateParams: datatypes.JSON([]byte(`{"installRoot":"/data/mysql","dataRoot":"/data/mysql/data","serviceName":"mysqld","port":"3306","rootPassword":"Secret_123"}`)),
	}
	mysqlPkg := DeployPackage{
		Name:           "mysql",
		Version:        `8.0.39"; touch /tmp/omx_fixed_pkg_version; #`,
		ExecutionMode:  ExecutionModeFixed,
		TemplateCode:   TemplateCodeMySQLSystemd,
		SourceFileName: "mysql-8.0.39.tar.gz",
		SourceURL:      "https://files.internal.local/mysql-8.0.39.tar.gz",
	}
	if _, err := renderFixedTemplateScript(mysqlPkg, mysqlTask); err == nil {
		t.Fatal("render must reject shell substitution in package version")
	}

	redisTask := DeployTask{
		Action:         TaskActionInstall,
		TemplateParams: datatypes.JSON([]byte(`{"installRoot":"/data/redis","dataRoot":"/data/redis/data","serviceName":"redis","port":"6379"}`)),
	}
	redisPkg := DeployPackage{
		Name:           "redis",
		Version:        "7.2.5$(touch /tmp/omx_fixed_pkg_version)",
		ExecutionMode:  ExecutionModeFixed,
		TemplateCode:   TemplateCodeRedisSystemd,
		SourceFileName: "redis-7.2.5.tar.gz",
		SourceURL:      "https://files.internal.local/redis-7.2.5.tar.gz",
	}
	if _, err := renderFixedTemplateScript(redisPkg, redisTask); err == nil {
		t.Fatal("render must reject shell substitution in Redis fallback package version")
	}
}

func TestFixedTemplateAllowsSemverBuildMetadataPackageVersion(t *testing.T) {
	pkg := DeployPackage{
		Name:           "mysql",
		Version:        "8.0.39+build.5",
		ExecutionMode:  ExecutionModeFixed,
		TemplateCode:   TemplateCodeMySQLSystemd,
		SourceFileName: "mysql-8.0.39.tar.gz",
		SourceURL:      "https://files.internal.local/mysql-8.0.39.tar.gz",
	}
	task := DeployTask{
		Action:         TaskActionInstall,
		TemplateParams: datatypes.JSON([]byte(`{"installRoot":"/data/mysql","dataRoot":"/data/mysql/data","serviceName":"mysqld","port":"3306","rootPassword":"Secret_123"}`)),
	}
	script, err := renderFixedTemplateScript(pkg, task)
	if err != nil {
		t.Fatalf("valid semver build metadata must render, got %v", err)
	}
	if !strings.Contains(script, `MYSQL_VERSION="8.0.39+build.5"`) {
		t.Fatal("expected package version in rendered MySQL script")
	}
}

func TestFixedTemplateValidParamsForAllBuiltInTemplates(t *testing.T) {
	cases := []struct {
		code   string
		params map[string]any
	}{
		{TemplateCodeNginxSystemd, map[string]any{"installRoot": "/data/nginx", "serviceName": "nginx"}},
		{TemplateCodeMySQLSystemd, map[string]any{"installRoot": "/data/mysql", "dataRoot": "/data/mysql/data", "serviceName": "mysqld", "port": "3306", "rootPassword": "Secret_123"}},
		{TemplateCodeRedisSystemd, map[string]any{"installRoot": "/data/redis", "dataRoot": "/data/redis/data", "serviceName": "redis", "port": "6379", "requirePassword": "Redis_123"}},
		{TemplateCodeMinIOSystemd, map[string]any{"installRoot": "/data/minio", "dataRoot": "/data/minio/data", "serviceName": "minio", "apiPort": "9000", "consolePort": "9001", "rootUser": "minioadmin", "rootPassword": "Minio_123"}},
		{TemplateCodeHarborOffline, map[string]any{"installRoot": "/data/harbor", "dataRoot": "/data/harbor/data", "hostname": "harbor.local", "httpPort": "8088", "adminPassword": "Harbor_123"}},
		{TemplateCodeJDKSystemd, map[string]any{"installRoot": "/data/jdk", "serviceName": "jdk", "mainClass": "com.example.Main", "appArgs": "--port 8080"}},
		{TemplateCodePrometheusSystemd, map[string]any{"installRoot": "/data/prometheus", "dataRoot": "/data/prometheus/data", "serviceName": "prometheus", "webPort": "9090", "storageRetentionPolicy": "30d", "scrapeConfigFile": "/etc/prometheus/custom.yml"}},
	}
	if len(cases) != 7 {
		t.Fatalf("expected all seven built-in fixed templates, got %d", len(cases))
	}
	for _, tc := range cases {
		tc.params["action"] = TaskActionInstall
		if err := validateFixedTemplateParams(tc.code, tc.params); err != nil {
			t.Fatalf("%s valid defaults must pass, got %v", tc.code, err)
		}
	}
}

func TestFixedTemplateRejectsUnsafeMetadataForAllBuiltInTemplates(t *testing.T) {
	cases := []struct {
		code   string
		params map[string]any
	}{
		{TemplateCodeNginxSystemd, map[string]any{"installRoot": "/data/nginx", "serviceName": "nginx$(id)"}},
		{TemplateCodeMySQLSystemd, map[string]any{"installRoot": "/data/mysql", "dataRoot": "/data/mysql/data", "serviceName": "mysqld", "port": "3306$(id)"}},
		{TemplateCodeRedisSystemd, map[string]any{"installRoot": "/data/redis", "dataRoot": "/data/redis/data", "serviceName": "redis", "port": "6379", "version": "7.2.5$(id)"}},
		{TemplateCodeMinIOSystemd, map[string]any{"installRoot": "/data/minio", "dataRoot": "/data/minio/data", "serviceName": "minio", "apiPort": "9000", "consolePort": "9001", "rootUser": "minio$(id)", "rootPassword": "Minio_123"}},
		{TemplateCodeHarborOffline, map[string]any{"installRoot": "/data/harbor", "dataRoot": "/data/harbor/data", "hostname": "harbor$(id)", "httpPort": "8088", "adminPassword": "Harbor_123"}},
		{TemplateCodeJDKSystemd, map[string]any{"installRoot": "/data/jdk", "serviceName": "jdk", "mainClass": "com.example.Main", "appArgs": "--port $(id)"}},
		{TemplateCodePrometheusSystemd, map[string]any{"installRoot": "/data/prometheus", "dataRoot": "/data/prometheus/data", "serviceName": "prometheus", "webPort": "9090", "storageRetentionPolicy": "30d$(id)", "scrapeConfigFile": "/etc/prometheus/custom.yml"}},
	}
	if len(cases) != 7 {
		t.Fatalf("expected all seven built-in fixed templates, got %d", len(cases))
	}
	for _, tc := range cases {
		tc.params["action"] = TaskActionInstall
		if err := validateFixedTemplateParams(tc.code, tc.params); err == nil {
			t.Fatalf("%s must reject shell substitution metadata", tc.code)
		}
	}
}

func TestFixedTemplateRedisPasswordStaysOpaque(t *testing.T) {
	password := `p@ss;$(id)`
	params := map[string]any{
		"action":          TaskActionInstall,
		"installRoot":     "/data/redis",
		"dataRoot":        "/data/redis/data",
		"serviceName":     "redis",
		"port":            "6379",
		"requirePassword": password,
	}
	if err := validateFixedTemplateParams(TemplateCodeRedisSystemd, params); err != nil {
		t.Fatalf("shell-safe password must be accepted, got %v", err)
	}
	pkg := DeployPackage{
		Name:           "redis",
		Version:        "7.2.5",
		ExecutionMode:  ExecutionModeFixed,
		TemplateCode:   TemplateCodeRedisSystemd,
		SourceFileName: "redis-7.2.5.tar.gz",
		SourceURL:      "https://files.internal.local/redis-7.2.5.tar.gz",
	}
	task := DeployTask{
		Action:         TaskActionInstall,
		TemplateParams: datatypes.JSON([]byte(`{"installRoot":"/data/redis","dataRoot":"/data/redis/data","serviceName":"redis","port":"6379","requirePassword":"p@ss;$(id)"}`)),
	}
	script, err := renderFixedTemplateScript(pkg, task)
	if err != nil {
		t.Fatalf("render redis with opaque password: %v", err)
	}
	if !strings.Contains(script, `REDIS_PASSWORD='p@ss;$(id)'`) {
		t.Fatal("password must be transported through a single-quoted shell variable")
	}
	if strings.Contains(script, "requirepass p@ss") {
		t.Fatal("password must not be interpolated raw into redis.conf")
	}
}

// Snapshot validation must use the effective task/step action rather than an
// optional action key inside the parameter map.
func TestFixedTemplateSnapshotValidationUsesEffectiveUninstallAction(t *testing.T) {
	pkg := DeployPackage{ExecutionMode: ExecutionModeFixed, TemplateCode: TemplateCodeNginxSystemd}
	params := map[string]any{"installRoot": "/", "serviceName": "nginx"}
	if err := validateStepSnapshotParams(TemplateStepTypePackage, pkg, params, TaskActionUninstall); err == nil {
		t.Fatal("snapshot validation must reject dangerous uninstall root using the effective action")
	}
}

// Persisted tasks created before the gate must fail at render, including an
// uninstall whose root would otherwise reach rm -rf.
func TestFixedTemplateRenderRejectsUnsafePersistedUninstall(t *testing.T) {
	pkg := DeployPackage{
		Name:           "nginx",
		Version:        "1.25.3",
		ExecutionMode:  ExecutionModeFixed,
		TemplateCode:   TemplateCodeNginxSystemd,
		SourceFileName: "nginx-1.25.3.tar.gz",
		SourceURL:      "https://files.internal.local/nginx-1.25.3.tar.gz",
	}
	for _, root := range []string{"/", "/etc", "/data/nginx/../../etc"} {
		task := DeployTask{
			Action:         TaskActionUninstall,
			TemplateParams: datatypes.JSON([]byte(`{"installRoot":"` + root + `","serviceName":"nginx"}`)),
		}
		if _, err := renderFixedTemplateScript(pkg, task); err == nil {
			t.Fatalf("render must reject persisted uninstall root %q", root)
		}
	}
}


// Reviewer-only Go overlay. It generates scripts; it never executes commands.
func TestCodexReviewRedisGeneratedConfig(t *testing.T) {
    pkg := DeployPackage{Name: "redis", Version: "7.2.5", ExecutionMode: ExecutionModeFixed,
        TemplateCode: TemplateCodeRedisSystemd, SourceFileName: "redis-7.2.5.tar.gz",
        SourceURL: "https://files.internal.local/redis-7.2.5.tar.gz"}
    for _, password := range []string{"", "safe-acceptance-password"} {
        t.Run(map[bool]string{true:"with-password",false:"without-password"}[password!=""], func(t *testing.T) {
            params := `{"installRoot":"/data/redis","dataRoot":"/data/redis/data","serviceName":"redis","port":"6379","requirePassword":"`+password+`"}`
            task := DeployTask{Action:TaskActionInstall, TemplateParams:datatypes.JSON([]byte(params))}
            script, err := renderFixedTemplateScript(pkg,task)
            if err != nil { t.Fatal(err) }
            start := strings.Index(script, `cat > "$INSTALL_ROOT/conf/redis.conf" <<EOF`)
            if start < 0 { t.Fatal("config heredoc missing") }
            body := script[start:]
            end := strings.Index(body,"\nEOF\n")
            if end < 0 { t.Fatal("config heredoc terminator missing") }
            config := body[:end]
            for _, line := range []string{"\nport 6379\n", "\ndir /data/redis/data\n", "\nlogfile /data/redis/logs/redis.log\n"} {
                if !strings.Contains(config,line) { t.Errorf("generated config lacks %q; actual=%q",line,config) }
            }
            if strings.Contains(config,"printf ") || strings.Contains(config,"if [") {
                t.Error("shell password writer was placed inside redis.conf heredoc")
            }
            if strings.Contains(script,"\nEOF\n/data/redis\n") {
                t.Error("installation directory rendered as a bare executable command after heredoc")
            }
        })
    }
}

func TestCodexReviewRepeatedSlashUninstallRoot(t *testing.T) {
    pkg := DeployPackage{Name:"nginx",Version:"1.25.3",ExecutionMode:ExecutionModeFixed,
        TemplateCode:TemplateCodeNginxSystemd,SourceFileName:"nginx.tar.gz",
        SourceURL:"https://files.internal.local/nginx.tar.gz"}
    for _, path := range []string{"//etc","/usr//local"} {
        t.Run(strings.ReplaceAll(path,"/","_"),func(t *testing.T){
            task := DeployTask{Action:TaskActionUninstall,TemplateParams:datatypes.JSON([]byte(`{"installRoot":"`+path+`","serviceName":"nginx"}`))}
            script,err := renderFixedTemplateScript(pkg,task)
            if err == nil {
                t.Fatalf("dangerous POSIX alias %q accepted for uninstall (delete command present=%t); script was not executed",path,strings.Contains(script,"rm -rf"))
            }
        })
    }
}
