#!/usr/bin/env bash
#
# fetch-shopify.sh — mirror a Shopify storefront to local HTML for review.
#
# Usage:
#   ./scripts/fetch-shopify.sh <storefront-url> [output-dir]
#
# For password-gated stores (most staging), put SHOPIFY_PASSWORD in your
# .env or export it. The script will auto-load .env if present.
#
# Output dir defaults to tmp/shopify-mirror-<timestamp>/. After the mirror
# finishes, feed it straight to:
#
#   corpus review --batch <output-dir>
#
# Why bash and not Go: wget already does mirroring + cookie handling well,
# and reimplementing it in Go is busywork. Keep this thin.

set -euo pipefail

# Auto-load .env so SHOPIFY_PASSWORD is available without an explicit export.
if [ -f .env ]; then
    set -a
    # shellcheck disable=SC1091
    source .env
    set +a
fi

URL="${1:-}"
OUTDIR="${2:-}"
PASSWORD="${SHOPIFY_PASSWORD:-}"

if [ -z "$URL" ]; then
    cat >&2 <<EOF
usage: $0 <storefront-url> [output-dir]

For password-gated stores: set SHOPIFY_PASSWORD in .env or export it.

Examples:
  $0 https://example.myshopify.com
  $0 https://example.myshopify.com tmp/shopify-staging/
  SHOPIFY_PASSWORD=secret $0 https://example.myshopify.com
EOF
    exit 1
fi

if ! command -v wget >/dev/null 2>&1; then
    echo "error: wget is required (brew install wget)" >&2
    exit 1
fi

if [ -z "$OUTDIR" ]; then
    OUTDIR="tmp/shopify-mirror-$(date +%Y%m%d-%H%M%S)"
fi

mkdir -p "$OUTDIR"

# Extract host (strip scheme and trailing path).
HOST="$(echo "$URL" | sed -E 's|^https?://||' | sed -E 's|/.*$||')"
if [ -z "$HOST" ]; then
    echo "error: could not parse host from URL: $URL" >&2
    exit 1
fi

# Common wget mirror flags. Reject binary/asset extensions so we only get text.
COMMON_FLAGS=(
    --mirror
    --convert-links
    --adjust-extension
    --page-requisites
    --no-parent
    --reject "jpg,jpeg,png,gif,webp,woff,woff2,ttf,svg,ico,css,js,mp4,mov"
    --domains "$HOST"
    --no-verbose
)

cd "$OUTDIR"

if [ -n "$PASSWORD" ]; then
    echo "→ authenticating to $HOST..."
    wget --quiet --save-cookies cookies.txt \
        --keep-session-cookies \
        --post-data "form_type=storefront_password&utf8=%E2%9C%93&password=$PASSWORD" \
        "$URL/password" \
        -O /dev/null

    # Sanity check: fetch the homepage and look for the password gate.
    HOMEPAGE="$(wget --quiet --load-cookies cookies.txt -O - "$URL" | head -200 || true)"
    if echo "$HOMEPAGE" | grep -qiE 'opening soon|password.*required|enter.*password'; then
        echo "error: password authentication failed (still seeing password gate)" >&2
        echo "  check SHOPIFY_PASSWORD in .env" >&2
        exit 2
    fi

    echo "→ mirroring (authenticated)..."
    wget --load-cookies cookies.txt "${COMMON_FLAGS[@]}" "$URL"
else
    echo "→ mirroring (unauthenticated)..."
    wget "${COMMON_FLAGS[@]}" "$URL"
fi

cd - > /dev/null

HTML_COUNT="$(find "$OUTDIR" -name '*.html' | wc -l | tr -d ' ')"
echo ""
echo "done: $HTML_COUNT HTML file(s) in $OUTDIR"
echo ""
echo "next:"
echo "  corpus review --batch $OUTDIR"
