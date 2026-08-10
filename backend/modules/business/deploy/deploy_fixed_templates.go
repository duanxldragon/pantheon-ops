package deploy

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"gorm.io/datatypes"
)

func validateFixedTemplateDefinition(templateCode string) error {
	switch strings.TrimSpace(templateCode) {
	case "", TemplateCodeNginxSystemd, TemplateCodeMySQLSystemd, TemplateCodeRedisSystemd, TemplateCodeMinIOSystemd, TemplateCodeHarborOffline:
		return nil
	default:
		return errors.New("deploypackage.template_invalid")
	}
}

func validateFixedTemplateParams(templateCode string, templateParams map[string]any) error {
	action := normalizeTaskAction(anyToString(templateParams["action"]))
	if action == "" {
		action = TaskActionInstall
	}
	switch strings.TrimSpace(templateCode) {
	case TemplateCodeNginxSystemd:
		return requireTemplateParamsForAction(action, templateParams, "installRoot", "serviceName")
	case TemplateCodeMySQLSystemd:
		return requireTemplateParamsForAction(action, templateParams, "installRoot", "dataRoot", "serviceName", "port", "rootPassword")
	case TemplateCodeRedisSystemd:
		return requireTemplateParamsForAction(action, templateParams, "installRoot", "dataRoot", "serviceName", "port")
	case TemplateCodeMinIOSystemd:
		return requireTemplateParamsForAction(action, templateParams, "installRoot", "dataRoot", "serviceName", "apiPort", "consolePort", "rootUser", "rootPassword")
	case TemplateCodeHarborOffline:
		return requireTemplateParamsForAction(action, templateParams, "installRoot", "dataRoot", "hostname", "httpPort", "adminPassword")
	default:
		return nil
	}
}

func requireTemplateParamsForAction(action string, templateParams map[string]any, keys ...string) error {
	if action == TaskActionUninstall {
		return nil
	}
	for _, key := range keys {
		if strings.TrimSpace(anyToString(templateParams[key])) == "" {
			return errors.New(errDeployTaskTemplateParamsInvalid)
		}
	}
	return nil
}

func renderFixedTemplateScript(pkg DeployPackage, task DeployTask) (string, error) {
	switch strings.TrimSpace(pkg.TemplateCode) {
	case TemplateCodeNginxSystemd:
		return renderNginxSystemdScript(pkg, task)
	case TemplateCodeMySQLSystemd:
		return renderMySQLSystemdScript(pkg, task)
	case TemplateCodeRedisSystemd:
		return renderRedisSystemdScript(pkg, task)
	case TemplateCodeMinIOSystemd:
		return renderMinIOSystemdScript(pkg, task)
	case TemplateCodeHarborOffline:
		return renderHarborOfflineScript(pkg, task)
	default:
		return "", errors.New(errDeployTaskTemplateInvalid)
	}
}

func buildServiceStopAndDrainScript(serviceName string, processPattern string) string {
	serviceName = strings.TrimSpace(serviceName)
	processPattern = strings.TrimSpace(processPattern)
	lines := []string{
		`if command -v systemctl >/dev/null 2>&1; then`,
		fmt.Sprintf(`  systemctl stop %q >/dev/null 2>&1 || true`, serviceName),
		`fi`,
	}
	if processPattern != "" {
		lines = append(lines,
			`if command -v pgrep >/dev/null 2>&1; then`,
			fmt.Sprintf(`  for _ in 1 2 3 4 5; do`),
			fmt.Sprintf(`    if ! pgrep -f %q >/dev/null 2>&1; then`, processPattern),
			`      break`,
			`    fi`,
			fmt.Sprintf(`    pkill -f %q >/dev/null 2>&1 || true`, processPattern),
			`    sleep 1`,
			`  done`,
			`fi`,
		)
	}
	return strings.Join(lines, "\n")
}

func renderMySQLSystemdScript(pkg DeployPackage, task DeployTask) (string, error) {
	params := decodeJSONMap(task.TemplateParams)
	action := normalizeTaskAction(task.Action)
	installRoot := templateParamString(params, "installRoot", "/data/mysql")
	dataRoot := templateParamString(params, "dataRoot", filepath.ToSlash(filepath.Join(installRoot, "data")))
	serviceName := templateParamString(params, "serviceName", "mysqld")
	port := templateParamString(params, "port", "3306")
	rootPassword := templateParamString(params, "rootPassword", "")
	if action != TaskActionUninstall && strings.TrimSpace(rootPassword) == "" {
		return "", errors.New(errDeployTaskTemplateParamsInvalid)
	}
	if action == TaskActionUninstall {
		return fmt.Sprintf(`set -e
SERVICE_NAME="%s"
INSTALL_ROOT="%s"
DATA_ROOT="%s"
systemctl stop "${SERVICE_NAME}" || true
systemctl disable "${SERVICE_NAME}" || true
rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
systemctl daemon-reload
rm -rf "$INSTALL_ROOT" "$DATA_ROOT"
id mysql >/dev/null 2>&1 && userdel mysql || true
getent group mysql >/dev/null 2>&1 && groupdel mysql || true
echo "MySQL uninstall completed"
`, serviceName, installRoot, dataRoot), nil
	}
	archiveName, downloadScript, err := buildSourceFetchScript(pkg, "", "$PKG_DIR")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`set -e
MYSQL_VERSION="%s"
INSTALL_ROOT="%s"
DATA_ROOT="%s"
SERVICE_NAME="%s"
MYSQL_PORT="%s"
ROOT_PASSWORD="%s"
PKG_DIR="/tmp/mysql-${MYSQL_VERSION}"
STAGE_DIR="$PKG_DIR/stage"
ARCHIVE_NAME="%s"
mkdir -p "$PKG_DIR" "$STAGE_DIR" "$INSTALL_ROOT" "$DATA_ROOT"
%s
%s
rm -rf "$STAGE_DIR"/*
tar -xf "$PKG_DIR/$ARCHIVE_NAME" -C "$STAGE_DIR"
SRC_DIR="$(find "$STAGE_DIR" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
if [ -z "$SRC_DIR" ]; then
  SRC_DIR="$STAGE_DIR"
fi
if command -v apt-get >/dev/null 2>&1; then
  apt-get update
  DEBIAN_FRONTEND=noninteractive apt-get install -y libaio1 libncurses5 xz-utils rsync
elif command -v yum >/dev/null 2>&1; then
  yum install -y libaio ncurses-compat-libs xz rsync
fi
getent group mysql >/dev/null 2>&1 || groupadd mysql
id mysql >/dev/null 2>&1 || useradd --system --gid mysql --home-dir "$INSTALL_ROOT" --shell /sbin/nologin mysql
rsync -a --delete "$SRC_DIR"/ "$INSTALL_ROOT"/
mkdir -p "$INSTALL_ROOT/conf" "$INSTALL_ROOT/run" "$INSTALL_ROOT/logs" "$DATA_ROOT"
cat > "$INSTALL_ROOT/conf/my.cnf" <<EOF
[mysqld]
basedir=%s
datadir=%s
socket=%s/run/mysql.sock
pid-file=%s/run/mysql.pid
port=%s
user=mysql
log-error=%s/logs/error.log
bind-address=0.0.0.0
symbolic-links=0
EOF
chown -R mysql:mysql "$INSTALL_ROOT" "$DATA_ROOT"
if [ ! -d "$DATA_ROOT/mysql" ]; then
  "$INSTALL_ROOT/bin/mysqld" --defaults-file="$INSTALL_ROOT/conf/my.cnf" --initialize-insecure --user=mysql
fi
cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=MySQL Server
After=network.target

[Service]
Type=simple
User=mysql
Group=mysql
ExecStart=%s/bin/mysqld --defaults-file=%s/conf/my.cnf
LimitNOFILE=65535
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
systemctl restart "${SERVICE_NAME}"
sleep 8
"$INSTALL_ROOT/bin/mysqladmin" --protocol=socket --socket="$INSTALL_ROOT/run/mysql.sock" -uroot password "$ROOT_PASSWORD" >/dev/null 2>&1 || true
systemctl status "${SERVICE_NAME}" --no-pager
echo "MySQL installed at $INSTALL_ROOT"
`, pkg.Version, installRoot, dataRoot, serviceName, port, rootPassword, archiveName, downloadScript, buildServiceStopAndDrainScript(serviceName, filepath.ToSlash(filepath.Join(installRoot, "bin", "mysqld"))), installRoot, dataRoot, installRoot, installRoot, port, installRoot, installRoot, installRoot), nil
}

func renderRedisSystemdScript(pkg DeployPackage, task DeployTask) (string, error) {
	params := decodeJSONMap(task.TemplateParams)
	action := normalizeTaskAction(task.Action)
	installRoot := templateParamString(params, "installRoot", "/data/redis")
	dataRoot := templateParamString(params, "dataRoot", filepath.ToSlash(filepath.Join(installRoot, "data")))
	serviceName := templateParamString(params, "serviceName", "redis")
	port := templateParamString(params, "port", "6379")
	requirePassword := templateParamString(params, "requirePassword", "")
	if action == TaskActionUninstall {
		return fmt.Sprintf(`set -e
SERVICE_NAME="%s"
INSTALL_ROOT="%s"
DATA_ROOT="%s"
systemctl stop "${SERVICE_NAME}" || true
systemctl disable "${SERVICE_NAME}" || true
rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
systemctl daemon-reload
rm -rf "$INSTALL_ROOT" "$DATA_ROOT"
id redis >/dev/null 2>&1 && userdel redis || true
echo "Redis uninstall completed"
`, serviceName, installRoot, dataRoot), nil
	}
	archiveName, downloadScript, err := buildSourceFetchScript(pkg, fmt.Sprintf("https://download.redis.io/releases/redis-%s.tar.gz", pkg.Version), "$PKG_DIR")
	if err != nil {
		return "", err
	}
	passwordLine := ""
	if requirePassword != "" {
		passwordLine = fmt.Sprintf("requirepass %s\nmasterauth %s\n", requirePassword, requirePassword)
	}
	return fmt.Sprintf(`set -e
REDIS_VERSION="%s"
INSTALL_ROOT="%s"
DATA_ROOT="%s"
SERVICE_NAME="%s"
REDIS_PORT="%s"
PKG_DIR="/tmp/redis-${REDIS_VERSION}"
STAGE_DIR="$PKG_DIR/stage"
ARCHIVE_NAME="%s"
mkdir -p "$PKG_DIR" "$STAGE_DIR" "$INSTALL_ROOT/bin" "$INSTALL_ROOT/conf" "$INSTALL_ROOT/logs" "$DATA_ROOT"
%s
%s
rm -rf "$STAGE_DIR"/*
tar -xf "$PKG_DIR/$ARCHIVE_NAME" -C "$STAGE_DIR"
SRC_DIR="$(find "$STAGE_DIR" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
if command -v apt-get >/dev/null 2>&1; then
  apt-get update
  DEBIAN_FRONTEND=noninteractive apt-get install -y build-essential tcl
elif command -v yum >/dev/null 2>&1; then
  yum groupinstall -y "Development Tools" || yum install -y gcc make
fi
cd "$SRC_DIR"
make -j"$(nproc)"
cp src/redis-server src/redis-cli src/redis-benchmark "$INSTALL_ROOT/bin/"
id redis >/dev/null 2>&1 || useradd --system --home-dir "$INSTALL_ROOT" --shell /sbin/nologin redis
cat > "$INSTALL_ROOT/conf/redis.conf" <<EOF
bind 0.0.0.0
port %s
dir %s
logfile %s/logs/redis.log
daemonize no
appendonly yes
protected-mode no
%sEOF
chown -R redis:redis "$INSTALL_ROOT" "$DATA_ROOT"
cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=Redis In-Memory Data Store
After=network.target

[Service]
Type=simple
User=redis
Group=redis
ExecStart=%s/bin/redis-server %s/conf/redis.conf
ExecStop=%s/bin/redis-cli -p %s shutdown
Restart=always

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
systemctl restart "${SERVICE_NAME}"
systemctl status "${SERVICE_NAME}" --no-pager
echo "Redis installed at $INSTALL_ROOT"
`, pkg.Version, installRoot, dataRoot, serviceName, port, archiveName, downloadScript, buildServiceStopAndDrainScript(serviceName, filepath.ToSlash(filepath.Join(installRoot, "bin", "redis-server"))), port, dataRoot, installRoot, passwordLine, installRoot, installRoot, installRoot, port), nil
}

func renderMinIOSystemdScript(pkg DeployPackage, task DeployTask) (string, error) {
	params := decodeJSONMap(task.TemplateParams)
	action := normalizeTaskAction(task.Action)
	installRoot := templateParamString(params, "installRoot", "/data/minio")
	dataRoot := templateParamString(params, "dataRoot", filepath.ToSlash(filepath.Join(installRoot, "data")))
	serviceName := templateParamString(params, "serviceName", "minio")
	apiPort := templateParamString(params, "apiPort", "9000")
	consolePort := templateParamString(params, "consolePort", "9001")
	rootUser := templateParamString(params, "rootUser", "minioadmin")
	rootPassword := templateParamString(params, "rootPassword", "")
	if action != TaskActionUninstall && strings.TrimSpace(rootPassword) == "" {
		return "", errors.New(errDeployTaskTemplateParamsInvalid)
	}
	if action == TaskActionUninstall {
		return fmt.Sprintf(`set -e
SERVICE_NAME="%s"
INSTALL_ROOT="%s"
DATA_ROOT="%s"
systemctl stop "${SERVICE_NAME}" || true
systemctl disable "${SERVICE_NAME}" || true
rm -f "/etc/systemd/system/${SERVICE_NAME}.service" "/etc/default/${SERVICE_NAME}"
systemctl daemon-reload
rm -rf "$INSTALL_ROOT" "$DATA_ROOT"
id minio >/dev/null 2>&1 && userdel minio || true
echo "MinIO uninstall completed"
`, serviceName, installRoot, dataRoot), nil
	}
	archiveName, downloadScript, err := buildSourceFetchScript(pkg, "https://dl.minio.org.cn/server/minio/release/linux-amd64/minio", "$PKG_DIR")
	if err != nil {
		return "", err
	}
	extractScript := minioArtifactInstallScript(archiveName)
	return fmt.Sprintf(`set -e
INSTALL_ROOT="%s"
DATA_ROOT="%s"
SERVICE_NAME="%s"
API_PORT="%s"
CONSOLE_PORT="%s"
ROOT_USER="%s"
ROOT_PASSWORD="%s"
PKG_DIR="/tmp/%s"
ARCHIVE_NAME="%s"
mkdir -p "$PKG_DIR" "$INSTALL_ROOT/bin" "$DATA_ROOT"
%s
%s
%s
id minio >/dev/null 2>&1 || useradd --system --home-dir "$INSTALL_ROOT" --shell /sbin/nologin minio
cat > "/etc/default/${SERVICE_NAME}" <<EOF
MINIO_ROOT_USER=%s
MINIO_ROOT_PASSWORD=%s
MINIO_VOLUMES=%s
MINIO_OPTS=--address :%s --console-address :%s
EOF
chown -R minio:minio "$INSTALL_ROOT" "$DATA_ROOT"
chmod +x "$INSTALL_ROOT/bin/minio"
cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=MinIO Object Storage
After=network.target

[Service]
User=minio
Group=minio
EnvironmentFile=/etc/default/%s
ExecStart=%s/bin/minio server $MINIO_VOLUMES $MINIO_OPTS
Restart=always
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
systemctl restart "${SERVICE_NAME}"
systemctl status "${SERVICE_NAME}" --no-pager
echo "MinIO installed at $INSTALL_ROOT"
`, installRoot, dataRoot, serviceName, apiPort, consolePort, rootUser, rootPassword, serviceName, archiveName, downloadScript, extractScript, buildServiceStopAndDrainScript(serviceName, filepath.ToSlash(filepath.Join(installRoot, "bin", "minio"))), rootUser, rootPassword, dataRoot, apiPort, consolePort, serviceName, installRoot), nil
}

func renderHarborOfflineScript(pkg DeployPackage, task DeployTask) (string, error) {
	params := decodeJSONMap(task.TemplateParams)
	action := normalizeTaskAction(task.Action)
	installRoot := templateParamString(params, "installRoot", "/data/harbor")
	dataRoot := templateParamString(params, "dataRoot", filepath.ToSlash(filepath.Join(installRoot, "data")))
	hostname := templateParamString(params, "hostname", "")
	httpPort := templateParamString(params, "httpPort", "8088")
	adminPassword := templateParamString(params, "adminPassword", "")
	if action != TaskActionUninstall && (hostname == "" || adminPassword == "") {
		return "", errors.New(errDeployTaskTemplateParamsInvalid)
	}
	if action == TaskActionUninstall {
		return fmt.Sprintf(`set -e
INSTALL_ROOT="%s"
DATA_ROOT="%s"
if [ -f "$INSTALL_ROOT/installer/docker-compose.yml" ]; then
  (cd "$INSTALL_ROOT/installer" && (docker compose down -v || docker-compose down -v || true))
fi
rm -rf "$INSTALL_ROOT" "$DATA_ROOT"
echo "Harbor uninstall completed"
`, installRoot, dataRoot), nil
	}
	archiveName, downloadScript, err := buildSourceFetchScript(pkg, "", "$PKG_DIR")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`set -e
INSTALL_ROOT="%s"
DATA_ROOT="%s"
HARBOR_HOSTNAME="%s"
HTTP_PORT="%s"
ADMIN_PASSWORD="%s"
PKG_DIR="/tmp/harbor-installer"
ARCHIVE_NAME="%s"
mkdir -p "$PKG_DIR" "$INSTALL_ROOT" "$DATA_ROOT" "$INSTALL_ROOT/logs"
%s
rm -rf "$INSTALL_ROOT/installer"
mkdir -p "$INSTALL_ROOT/installer"
tar -xf "$PKG_DIR/$ARCHIVE_NAME" -C "$INSTALL_ROOT/installer" --strip-components=1
command -v docker >/dev/null 2>&1
(docker compose version >/dev/null 2>&1 || docker-compose version >/dev/null 2>&1)
cat > "$INSTALL_ROOT/installer/harbor.yml" <<EOF
hostname: $HARBOR_HOSTNAME
http:
  port: $HTTP_PORT
harbor_admin_password: $ADMIN_PASSWORD
data_volume: %s
log:
  level: info
  local:
    location: %s/logs
EOF
cd "$INSTALL_ROOT/installer"
./install.sh
echo "Harbor installed at $INSTALL_ROOT"
`, installRoot, dataRoot, hostname, httpPort, adminPassword, archiveName, downloadScript, dataRoot, installRoot), nil
}

func buildSourceFetchScript(pkg DeployPackage, fallbackURL string, targetDirExpr string) (string, string, error) {
	artifactName := strings.TrimSpace(pkg.SourceFileName)
	if artifactName == "" {
		if fallbackURL != "" {
			artifactName = filepath.Base(fallbackURL)
		} else {
			return "", "", errors.New(errDeployTaskPackageSourceMissing)
		}
	}
	if strings.TrimSpace(pkg.SourceObjectKey) != "" || strings.TrimSpace(pkg.SourceURL) != "" {
		if strings.TrimSpace(pkg.SourceURL) == "" {
			return "", "", errors.New(errDeployTaskPackageSourceMissing)
		}
		return artifactName, fmt.Sprintf(`SOURCE_URL="%s"
curl --connect-timeout 15 --max-time 300 --retry 2 --retry-delay 2 -fsSL "$SOURCE_URL" -o %s/%s
`, pkg.SourceURL, targetDirExpr, artifactName), nil
	}
	if fallbackURL == "" {
		return "", "", errors.New(errDeployTaskPackageSourceMissing)
	}
	return artifactName, fmt.Sprintf(`curl --connect-timeout 15 --max-time 300 --retry 2 --retry-delay 2 -fsSL "%s" -o %s/%s
`, fallbackURL, targetDirExpr, artifactName), nil
}

func minioArtifactInstallScript(artifactName string) string {
	name := strings.ToLower(strings.TrimSpace(artifactName))
	if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz") || strings.HasSuffix(name, ".tar") || strings.HasSuffix(name, ".tar.xz") {
		return `rm -rf "$PKG_DIR/extract"
mkdir -p "$PKG_DIR/extract"
tar -xf "$PKG_DIR/$ARCHIVE_NAME" -C "$PKG_DIR/extract"
cp "$(find "$PKG_DIR/extract" -type f -name minio | head -n 1)" "$INSTALL_ROOT/bin/minio"
`
	}
	if strings.HasSuffix(name, ".zip") {
		return `rm -rf "$PKG_DIR/extract"
mkdir -p "$PKG_DIR/extract"
unzip -o "$PKG_DIR/$ARCHIVE_NAME" -d "$PKG_DIR/extract"
cp "$(find "$PKG_DIR/extract" -type f -name minio | head -n 1)" "$INSTALL_ROOT/bin/minio"
`
	}
	return `cp "$PKG_DIR/$ARCHIVE_NAME" "$INSTALL_ROOT/bin/minio"
`
}

func templateParamString(values map[string]any, key string, fallback string) string {
	value := strings.TrimSpace(anyToString(values[key]))
	if value == "" {
		return fallback
	}
	return value
}

func validateTemplateParamsForCode(executionMode string, templateCode string, templateConfigRaw datatypes.JSON, templateParams map[string]any) error {
	mode := strings.TrimSpace(executionMode)
	code := strings.TrimSpace(templateCode)
	if mode != ExecutionModeFixed || code == "" {
		return nil
	}
	_ = templateConfigRaw
	return validateFixedTemplateParams(code, templateParams)
}
