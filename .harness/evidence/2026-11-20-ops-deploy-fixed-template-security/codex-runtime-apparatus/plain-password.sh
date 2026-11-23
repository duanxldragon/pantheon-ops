set -e
REDIS_VERSION="7.2.5"
INSTALL_ROOT="/tmp/omx_accept_20261121_codex_t02a/redis-config/plain-password"
DATA_ROOT="/tmp/omx_accept_20261121_codex_t02a/redis-config/plain-password/data"
SERVICE_NAME="omx-redis-config"
REDIS_PORT="16380"
PKG_DIR="/tmp/redis-${REDIS_VERSION}"
STAGE_DIR="$PKG_DIR/stage"
ARCHIVE_NAME="redis-7.2.5.tar.gz"
REDIS_PASSWORD='safe_acceptance_pw'
REDIS_CONFIG_PASSWORD='"safe_acceptance_pw"'

mkdir -p "$PKG_DIR" "$STAGE_DIR" "$INSTALL_ROOT/bin" "$INSTALL_ROOT/conf" "$INSTALL_ROOT/logs" "$DATA_ROOT"
SOURCE_URL='https://files.internal.local/redis-7.2.5.tar.gz'
curl --connect-timeout 15 --max-time 300 --retry 2 --retry-delay 2 -fsSL "$SOURCE_URL" -o $PKG_DIR/'redis-7.2.5.tar.gz'

if command -v systemctl >/dev/null 2>&1; then
  systemctl stop "omx-redis-config" >/dev/null 2>&1 || true
fi
if command -v pgrep >/dev/null 2>&1; then
  for _ in 1 2 3 4 5; do
    if ! pgrep -f "/tmp/omx_accept_20261121_codex_t02a/redis-config/plain-password/bin/redis-server" >/dev/null 2>&1; then
      break
    fi
    pkill -f "/tmp/omx_accept_20261121_codex_t02a/redis-config/plain-password/bin/redis-server" >/dev/null 2>&1 || true
    sleep 1
  done
fi
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
port 16380
dir /tmp/omx_accept_20261121_codex_t02a/redis-config/plain-password/data
logfile /tmp/omx_accept_20261121_codex_t02a/redis-config/plain-password/logs/redis.log
daemonize no
appendonly yes
protected-mode no
EOF
if [ -n "$REDIS_PASSWORD" ]; then
  printf 'requirepass %s\nmasterauth %s\n' "$REDIS_CONFIG_PASSWORD" "$REDIS_CONFIG_PASSWORD" >> "$INSTALL_ROOT/conf/redis.conf"
fi

chown -R redis:redis "$INSTALL_ROOT" "$DATA_ROOT"
cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=Redis In-Memory Data Store
After=network.target

[Service]
Type=simple
User=redis
Group=redis
ExecStart=/tmp/omx_accept_20261121_codex_t02a/redis-config/plain-password/bin/redis-server /tmp/omx_accept_20261121_codex_t02a/redis-config/plain-password/conf/redis.conf
Restart=always

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
systemctl restart "${SERVICE_NAME}"
systemctl status "${SERVICE_NAME}" --no-pager
rm -rf "$PKG_DIR"
echo "Redis installed at $INSTALL_ROOT"
