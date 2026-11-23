package deploy

import (
	"encoding/json"
	"os"
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

func TestFixedTemplateRedisGeneratedConfig(t *testing.T) {
	pkg := DeployPackage{Name: "redis", Version: "7.2.5", ExecutionMode: ExecutionModeFixed,
		TemplateCode: TemplateCodeRedisSystemd, SourceFileName: "redis-7.2.5.tar.gz",
		SourceURL: "https://files.internal.local/redis-7.2.5.tar.gz"}
	for _, tc := range []struct{ name, password string }{
		{"without-password", ""}, {"with-password", "p@ss;$(id)'quoted"},
		{"plain-password", "safe_acceptance_pw"}, {"escaped-password", "p@ss\\word\"quoted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			params, err := json.Marshal(map[string]any{"installRoot": "/data/redis", "dataRoot": "/data/redis/data",
				"serviceName": "redis", "port": "6380", "requirePassword": tc.password})
			if err != nil {
				t.Fatal(err)
			}
			script, err := renderFixedTemplateScript(pkg, DeployTask{Action: TaskActionInstall, TemplateParams: datatypes.JSON(params)})
			if err != nil {
				t.Fatal(err)
			}
			_, body, ok := strings.Cut(script, `cat > "$INSTALL_ROOT/conf/redis.conf" <<EOF`)
			if !ok {
				t.Fatal("config heredoc missing")
			}
			config, after, ok := strings.Cut(body, "\nEOF\n")
			if !ok {
				t.Fatal("config heredoc terminator missing")
			}
			for _, line := range []string{"\nport 6380\n", "\ndir /data/redis/data\n", "\nlogfile /data/redis/logs/redis.log\n"} {
				if !strings.Contains(config, line) {
					t.Errorf("generated config lacks %q; actual=%q", line, config)
				}
			}
			if strings.Contains(config, "printf ") || strings.Contains(config, "if [") {
				t.Error("shell password writer must not be inside redis.conf")
			}
			if strings.HasPrefix(after, "/data/redis\n") {
				t.Error("installation directory must not be a bare executable command")
			}
			writer := `printf 'requirepass %s\nmasterauth %s\n' "$REDIS_CONFIG_PASSWORD" "$REDIS_CONFIG_PASSWORD"`
			if strings.Contains(after, writer) != (tc.password != "") {
				t.Error("password writer must follow config heredoc only when a password is provided")
			}
		})
	}
}

func TestFixedTemplateRejectsUninstallRootAliases(t *testing.T) {
	templates := []string{TemplateCodeNginxSystemd, TemplateCodeMySQLSystemd, TemplateCodeRedisSystemd,
		TemplateCodeMinIOSystemd, TemplateCodeHarborOffline, TemplateCodeJDKSystemd, TemplateCodePrometheusSystemd}
	for _, code := range templates {
		for _, root := range []string{"//etc", "/usr//local", "/usr///local/", "///", "//var/", "/etc////"} {
			for _, key := range []string{"installRoot", "dataRoot"} {
				t.Run(code+"/"+key+"/"+root, func(t *testing.T) {
					params := map[string]any{"installRoot": "/data/app", "dataRoot": "/data/app/data",
						"serviceName": "app", "action": TaskActionUninstall}
					params[key] = root
					if err := validateFixedTemplateParams(code, params); err == nil {
						t.Error("create validation must reject dangerous uninstall root alias")
					}
					// A legacy snapshot may claim install; the effective action remains uninstall.
					params["action"] = TaskActionInstall
					pkg := DeployPackage{Name: "app", Version: "1.0.0", ExecutionMode: ExecutionModeFixed, TemplateCode: code}
					if err := validateStepSnapshotParams(TemplateStepTypePackage, pkg, params, TaskActionUninstall); err == nil {
						t.Error("snapshot must reject root alias using effective uninstall action")
					}
					raw, err := json.Marshal(params)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := renderFixedTemplateScript(pkg, DeployTask{Action: TaskActionUninstall, TemplateParams: datatypes.JSON(raw)}); err == nil {
						t.Error("persisted uninstall must reject root alias before rendering")
					}
				})
			}
		}
	}
	if err := validateFixedUninstallRoot("/data//app/"); err != nil {
		t.Fatalf("safe path alias must retain existing compatibility: %v", err)
	}
}

func TestCodexExportRedisArtifacts(t *testing.T) {
    pkg := DeployPackage{Name:"redis",Version:"7.2.5",ExecutionMode:ExecutionModeFixed,
        TemplateCode:TemplateCodeRedisSystemd,SourceFileName:"redis-7.2.5.tar.gz",
        SourceURL:"https://files.internal.local/redis-7.2.5.tar.gz"}

    {
        task := DeployTask{Action:TaskActionInstall,TemplateParams:datatypes.JSON([]byte("{\"installRoot\": \"/tmp/omx_accept_20261121_codex_t02a/redis-config/without-password\", \"dataRoot\": \"/tmp/omx_accept_20261121_codex_t02a/redis-config/without-password/data\", \"serviceName\": \"omx-redis-config\", \"port\": \"16380\", \"requirePassword\": \"\"}"))}
        script,err := renderFixedTemplateScript(pkg,task)
        if err != nil { t.Fatal(err) }
        if err := os.WriteFile("D:\\workspace\\go\\pantheon-platform\\pantheon-ops\\.harness\\evidence\\2026-11-20-ops-deploy-fixed-template-security\\codex-runtime-apparatus\\without-password.sh",[]byte(script),0600); err != nil { t.Fatal(err) }
    }

    {
        task := DeployTask{Action:TaskActionInstall,TemplateParams:datatypes.JSON([]byte("{\"installRoot\": \"/tmp/omx_accept_20261121_codex_t02a/redis-config/plain-password\", \"dataRoot\": \"/tmp/omx_accept_20261121_codex_t02a/redis-config/plain-password/data\", \"serviceName\": \"omx-redis-config\", \"port\": \"16380\", \"requirePassword\": \"safe_acceptance_pw\"}"))}
        script,err := renderFixedTemplateScript(pkg,task)
        if err != nil { t.Fatal(err) }
        if err := os.WriteFile("D:\\workspace\\go\\pantheon-platform\\pantheon-ops\\.harness\\evidence\\2026-11-20-ops-deploy-fixed-template-security\\codex-runtime-apparatus\\plain-password.sh",[]byte(script),0600); err != nil { t.Fatal(err) }
    }

    {
        task := DeployTask{Action:TaskActionInstall,TemplateParams:datatypes.JSON([]byte("{\"installRoot\": \"/tmp/omx_accept_20261121_codex_t02a/redis-config/quoted-password\", \"dataRoot\": \"/tmp/omx_accept_20261121_codex_t02a/redis-config/quoted-password/data\", \"serviceName\": \"omx-redis-config\", \"port\": \"16380\", \"requirePassword\": \"p@ss;$(id)'quoted\"}"))}
        script,err := renderFixedTemplateScript(pkg,task)
        if err != nil { t.Fatal(err) }
        if err := os.WriteFile("D:\\workspace\\go\\pantheon-platform\\pantheon-ops\\.harness\\evidence\\2026-11-20-ops-deploy-fixed-template-security\\codex-runtime-apparatus\\quoted-password.sh",[]byte(script),0600); err != nil { t.Fatal(err) }
    }

    {
        task := DeployTask{Action:TaskActionInstall,TemplateParams:datatypes.JSON([]byte("{\"installRoot\": \"/tmp/omx_accept_20261121_codex_t02a/redis-config/escaped-password\", \"dataRoot\": \"/tmp/omx_accept_20261121_codex_t02a/redis-config/escaped-password/data\", \"serviceName\": \"omx-redis-config\", \"port\": \"16380\", \"requirePassword\": \"p@ss\\\\word\\\"quoted\"}"))}
        script,err := renderFixedTemplateScript(pkg,task)
        if err != nil { t.Fatal(err) }
        if err := os.WriteFile("D:\\workspace\\go\\pantheon-platform\\pantheon-ops\\.harness\\evidence\\2026-11-20-ops-deploy-fixed-template-security\\codex-runtime-apparatus\\escaped-password.sh",[]byte(script),0600); err != nil { t.Fatal(err) }
    }

}
