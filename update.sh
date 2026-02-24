#!/bin/bash
set -e
curl -sL https://github.com/cafedomingo/SKU_RM0004/releases/latest/download/display \
  -o /opt/uctronics-lcd/display
chmod +x /opt/uctronics-lcd/display
systemctl restart uctronics-display
echo "Updated successfully"
