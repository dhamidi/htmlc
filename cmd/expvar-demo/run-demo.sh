#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Start the demo server in a tmux session.
tmux new-session -d -s expvar-demo -x 220 -y 50 \
  "cd '$WORKSPACE' && go run ./cmd/expvar-demo/ 2>&1 | tee /tmp/expvar-demo.log"
sleep 2

echo "=== /debug/vars after startup ==="
curl -s http://localhost:9876/debug/vars | python3 -m json.tool
echo ""

echo "=== Performing 3 renders ==="
curl -s http://localhost:9876/render > /dev/null
curl -s http://localhost:9876/render > /dev/null
curl -s http://localhost:9876/render > /dev/null
echo ""

echo "=== /debug/vars after 3 renders ==="
curl -s http://localhost:9876/debug/vars | python3 -m json.tool
echo ""

echo "=== Toggling debug on ==="
curl -s http://localhost:9876/admin/debug/on
echo ""

echo "=== /debug/vars showing debug:1 ==="
curl -s http://localhost:9876/debug/vars | python3 -m json.tool
echo ""

echo "=== Demo server still running in tmux session 'expvar-demo' ==="
echo "    tmux attach -t expvar-demo"
echo "    tmux kill-session -t expvar-demo"
