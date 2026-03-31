#!/bin/bash
#
# Installs or updates the UCTRONICS LCD display driver on Raspberry Pi 4/5.
#
# Install:  curl -sL https://github.com/cafedomingo/SKU_RM0004/releases/latest/download/install.sh | sudo bash
# Update:   (same command)
#
# Idempotent — safe to re-run after partial failures.

set -euo pipefail

REPO="cafedomingo/SKU_RM0004"
INSTALL_DIR="/opt/uctronics-lcd"
BINARY="display"
SERVICE_NAME="uctronics-display.service"
SERVICE_PATH="/etc/systemd/system/${SERVICE_NAME}"

needs_reboot=false

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

# --- Boot config ---

get_boot_config_path() {
    local version
    version=$(grep "VERSION_ID" /etc/os-release | cut -d= -f2 | tr -d '"')
    if [ "${version}" -ge 12 ] 2>/dev/null; then
        echo "/boot/firmware/config.txt"
    else
        echo "/boot/config.txt"
    fi
}

configure_boot() {
    local pi_model="$1"
    local boot_config
    boot_config=$(get_boot_config_path)

    [ -f "$boot_config" ] || die "Boot config not found: ${boot_config}"

    # GPIO shutdown overlay
    if ! grep -q "gpio-shutdown,gpio_pin=4" "$boot_config"; then
        log "Adding GPIO shutdown overlay to ${boot_config}"
        if [ "$pi_model" = "pi5" ]; then
            echo "dtoverlay=gpio-shutdown,gpio_pin=4,active_low=1,gpio_pull=up,debounce=1000" >> "$boot_config"
        else
            echo "dtoverlay=gpio-shutdown,gpio_pin=4,active_low=1,gpio_pull=up" >> "$boot_config"
        fi
        needs_reboot=true
    fi

    # I2C with 400kHz baud rate (must use dtparam= prefix for baudrate to take effect)
    if grep -q "i2c_arm_baudrate=400000" "$boot_config"; then
        # Fix bare i2c_arm_baudrate lines (missing dtparam= prefix) left by older installs
        if grep -q "^i2c_arm_baudrate=" "$boot_config"; then
            sed -i '/^i2c_arm_baudrate=/d' "$boot_config"
            sed -i 's/^dtparam=i2c_arm=on.*/dtparam=i2c_arm=on,i2c_arm_baudrate=400000/' "$boot_config"
            needs_reboot=true
        fi
    elif grep -q "^#dtparam=i2c_arm=on" "$boot_config"; then
        sed -i "s/^#dtparam=i2c_arm=on.*/dtparam=i2c_arm=on,i2c_arm_baudrate=400000/" "$boot_config"
        needs_reboot=true
    elif grep -q "^dtparam=i2c_arm=on" "$boot_config"; then
        sed -i "s/^dtparam=i2c_arm=on.*/dtparam=i2c_arm=on,i2c_arm_baudrate=400000/" "$boot_config"
        needs_reboot=true
    else
        echo "" >> "$boot_config"
        echo "dtparam=i2c_arm=on,i2c_arm_baudrate=400000" >> "$boot_config"
        needs_reboot=true
    fi

    # i2c-dev kernel module (provides /dev/i2c-* device nodes for userspace access)
    if ! grep -q "^i2c-dev" /etc/modules; then
        log "Adding i2c-dev to /etc/modules"
        echo "i2c-dev" >> /etc/modules
        needs_reboot=true
    fi
}

# --- Binary install ---

install_binary() {
    mkdir -p "$INSTALL_DIR"

    # Developer path: use local binary if run from a repo clone
    if [ -f "./${BINARY}" ] && [ -f "./go.mod" ]; then
        log "Installing local ./${BINARY} to ${INSTALL_DIR}/${BINARY}"
        cp "./${BINARY}" "${INSTALL_DIR}/${BINARY}"
    else
        if [ -f "./go.mod" ]; then
            log "No local binary — downloading from release (run 'go build -o display ./cmd/display' first to install a local build)"
        else
            log "Downloading ${BINARY} from latest release"
        fi
        if ! curl -fsSL "https://github.com/${REPO}/releases/latest/download/${BINARY}" \
            -o "${INSTALL_DIR}/${BINARY}"; then
            die "Failed to download ${BINARY} from GitHub releases"
        fi
        if [ ! -s "${INSTALL_DIR}/${BINARY}" ]; then
            die "Downloaded ${BINARY} is empty"
        fi
    fi

    chmod +x "${INSTALL_DIR}/${BINARY}"
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
    systemctl enable "$SERVICE_NAME"
}

# --- Main ---

log "Starting install"

pi_model=$(detect_pi_model)
log "Detected Raspberry Pi: ${pi_model}"

configure_boot "$pi_model"

# --- Version check ---

latest=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep -o '"tag_name":"[^"]*"' | cut -d'"' -f4)
current=$("${INSTALL_DIR}/${BINARY}" -version 2>/dev/null || echo "none")
if [ "$latest" = "$current" ]; then
    log "Already up to date (${current})"
    exit 0
fi

service_was_running=false
if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    service_was_running=true
    log "Stopping ${SERVICE_NAME}"
    systemctl stop "$SERVICE_NAME"
fi

install_binary
install_service

if [ "$needs_reboot" = true ]; then
    log "Install complete. Reboot required for boot config changes — the display service will start automatically after reboot."
else
    log "Starting ${SERVICE_NAME}"
    systemctl start "$SERVICE_NAME"
    if [ "$service_was_running" = true ]; then
        log "Updated successfully"
    else
        log "Install complete"
    fi
fi
