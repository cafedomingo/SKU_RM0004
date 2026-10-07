#!/bin/bash
#
# Installs or updates the UCTRONICS LCD display driver on Raspberry Pi 4/5.
#
# Install:  curl -sL https://github.com/cafedomingo/SKU_RM0004/releases/latest/download/install.sh | sudo bash
# Update:   (same command)
#
# Idempotent. Reports needed boot config changes instead of making them.

set -euo pipefail

REPO="cafedomingo/SKU_RM0004"
INSTALL_DIR="/opt/uctronics-lcd"
BINARY="display"
VERSION_FILE="${INSTALL_DIR}/VERSION"

SERVICE_NAME="uctronics-display.service"
SERVICE_PATH="/etc/systemd/system/${SERVICE_NAME}"
MODULES_PATH="/etc/modules-load.d/uctronics-lcd.conf"

BOOT_CONFIG="/boot/firmware/config.txt"
[ -f "$BOOT_CONFIG" ] || BOOT_CONFIG="/boot/config.txt"

I2C_BUS="/sys/bus/i2c/devices/i2c-1"
I2C_HZ=400000
I2C_LINE="dtparam=i2c_arm=on,i2c_arm_baudrate=${I2C_HZ}"
I2C_PATTERN='^(dtparam=.*i2c_arm|i2c_arm_baudrate)'

SHUTDOWN_LINE="dtoverlay=gpio-shutdown,gpio_pin=4,active_low=1,gpio_pull=up"
SHUTDOWN_PATTERN='^dtoverlay=gpio-shutdown'

log() {
    echo "[$(hostname)] $*"
}

die() {
    log "ERROR: $*" >&2
    exit 1
}

if [ "$(id -u)" -ne 0 ]; then
    die "This script must be run as root (use sudo)"
fi

# --- Pi model detection ---

detect_pi_model() {
    local model
    model=$(tr -d '\0' < /proc/device-tree/model 2>/dev/null) || die "Cannot read /proc/device-tree/model"

    if [[ "$model" == *"Raspberry Pi 5"* ]]; then
        echo "pi5"
    elif [[ "$model" == *"Raspberry Pi 4"* ]]; then
        echo "pi4"
    else
        die "Unsupported model: ${model}. Requires Raspberry Pi 4 or 5."
    fi
}

# --- Boot config check ---

i2c_bus_hz() {
    od -An -tu4 --endian=big "${I2C_BUS}/of_node/clock-frequency" 2>/dev/null | tr -d ' ' || true
}

# Prints "N: line" for uncommented config lines matching an ERE.
config_lines_matching() {
    awk -v pat="$1" '{ l = $0; sub(/#.*/, "", l); gsub(/^[ \t]+|[ \t]+$/, "", l)
        if (l ~ pat) print NR ": " l }' "$BOOT_CONFIG" 2>/dev/null || true
}

# Collects needed config.txt lines (want) and existing lines that conflict.
want=()
conflicts=()
pending=()
i2c_ready=false
need_line() {
    local line="$1" pattern="$2" found=false entry
    while IFS= read -r entry; do
        [ -n "$entry" ] || continue
        if [ "${entry#*: }" = "$line" ]; then
            found=true
        else
            conflicts+=("$entry")
        fi
    done <<<"$(config_lines_matching "$pattern")"
    if [ "$found" = true ]; then
        pending+=("$line")
    else
        want+=("$line")
    fi
}

check_boot_config() {
    local pi_model="$1"

    if [ ! -e "$I2C_BUS" ]; then
        log "I2C is not enabled"
        need_line "$I2C_LINE" "$I2C_PATTERN"
    else
        i2c_ready=true
        local hz
        hz=$(i2c_bus_hz)
        if [ "$hz" != "$I2C_HZ" ]; then
            log "I2C bus runs at ${hz:-?} Hz; the display needs ${I2C_HZ} Hz"
            need_line "$I2C_LINE" "$I2C_PATTERN"
        fi
    fi

    if [ -z "$(find /proc/device-tree/ -maxdepth 4 -name 'shutdown_button@4' 2>/dev/null)" ]; then
        log "Shutdown overlay for the case's power button not detected"
        local line="$SHUTDOWN_LINE"
        [ "$pi_model" = "pi5" ] && line+=",debounce=1000"
        need_line "$line" "$SHUTDOWN_PATTERN"
    fi
}

# --- i2c-dev module ---

load_i2c_dev() {
    echo "i2c-dev" > "$MODULES_PATH"
    modprobe i2c-dev || log "Could not load i2c-dev now; it will load on next boot"
}

# --- Binary install ---

latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
        | grep '"tag_name"' | cut -d'"' -f4 || true
}

# Downloads beside the live binary so a failed download leaves it untouched.
binary_updated=false
install_binary() {
    mkdir -p "$INSTALL_DIR"
    local tmp="${INSTALL_DIR}/${BINARY}.new"
    local version

    # Developer path: use local binary if run from a repo clone
    if [ -f "./${BINARY}" ] && [ -f "./go.mod" ]; then
        log "Installing local ./${BINARY}"
        cp "./${BINARY}" "$tmp"
        version="local"
    else
        version=$(latest_version)
        if [ -n "$version" ] && [ "$version" = "$(cat "$VERSION_FILE" 2>/dev/null)" ] \
            && [ -x "${INSTALL_DIR}/${BINARY}" ]; then
            log "Binary already up to date (${version})"
            return
        fi
        if [ -f "./go.mod" ]; then
            log "No local binary, downloading from release (run 'go build -o display ./cmd/display' first to install a local build)"
        else
            log "Downloading ${BINARY} ${version:-latest} from release"
        fi
        curl -fsSL "https://github.com/${REPO}/releases/latest/download/${BINARY}" -o "$tmp" \
            || { rm -f "$tmp"; die "Failed to download ${BINARY} from GitHub releases"; }
        [ -s "$tmp" ] || { rm -f "$tmp"; die "Downloaded ${BINARY} is empty"; }
    fi

    chmod +x "$tmp"
    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    mv -f "$tmp" "${INSTALL_DIR}/${BINARY}"
    echo "$version" > "$VERSION_FILE"
    binary_updated=true
}

# --- Systemd service ---

install_service() {
    cat > "$SERVICE_PATH" <<EOF
[Unit]
Description=UCTRONICS LCD Display
After=multi-user.target

[Service]
ExecStart=${INSTALL_DIR}/${BINARY}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable --quiet "$SERVICE_NAME"
}

# --- Main ---

log "Starting install"

pi_model=$(detect_pi_model)
log "Detected Raspberry Pi: ${pi_model}"

check_boot_config "$pi_model"
load_i2c_dev
install_binary
install_service

if [ "$i2c_ready" = true ]; then
    if [ "$binary_updated" = true ] || ! systemctl is-active --quiet "$SERVICE_NAME"; then
        log "Starting ${SERVICE_NAME}"
        systemctl restart "$SERVICE_NAME"
    fi
else
    log "Not starting ${SERVICE_NAME} until I2C is enabled; it starts on boot"
fi

log "Install complete"
if [ ${#want[@]} -gt 0 ] || [ ${#pending[@]} -gt 0 ]; then
    echo
    log "Boot config changes needed in ${BOOT_CONFIG}:"
    if [ ${#conflicts[@]} -gt 0 ]; then
        log "Remove or comment out these lines:"
        printf '    line %s\n' "${conflicts[@]}"
    fi
    if [ ${#want[@]} -gt 0 ]; then
        log "Add these lines at the end of the file:"
        printf '    %s\n' "[all]" "${want[@]}"
    fi
    if [ ${#pending[@]} -gt 0 ]; then
        log "Already present, but not active yet (or inside a section that excludes this Pi):"
        printf '    %s\n' "${pending[@]}"
    fi
    log "Then reboot."
fi
