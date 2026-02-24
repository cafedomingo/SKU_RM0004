#!/bin/bash
set -e
systemctl stop uctronics-display
curl -sL https://github.com/cafedomingo/SKU_RM0004/releases/latest/download/display \
  -o /opt/uctronics-lcd/display
chmod +x /opt/uctronics-lcd/display
systemctl start uctronics-display
echo "Updated successfully"
