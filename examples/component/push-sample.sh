#!/usr/bin/env bash
# Build the sample OCM component and push it to GHCR.
#
# Requirements:
#   ocm CLI  https://github.com/open-component-model/ocm/releases
#
# GHCR authentication:
#   Create a PAT with 'write:packages' scope at https://github.com/settings/tokens
#   Then log in:
#     echo "$GHCR_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
#   Or configure an OCM credentials file:
#     ocm config add credentials ghcr.io username=YOUR_USER password=$GHCR_TOKEN
#
# Usage:
#   cd examples/component
#   bash push-sample.sh
#
# To push to a different registry:
#   TARGET=ghcr.io/your-org/your-repo bash push-sample.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="${TARGET:-ghcr.io/jakobmoellerdev/model-server/models}"
COMPONENT="example.org/tiny-model"
VERSION="1.0.0"
ARCHIVE="$SCRIPT_DIR/transport-archive"

cd "$SCRIPT_DIR"

echo "=== Building OCM component archive ==="
rm -rf "$ARCHIVE"
ocm add componentversions \
  --create \
  --file "./$ARCHIVE" \
  component-constructor.yaml
echo "Built: $ARCHIVE"
echo

echo "=== Verifying CTF contents ==="
ocm get componentversions "ctf::$ARCHIVE"
echo

echo "=== Pushing to $TARGET ==="
ocm transfer componentversions \
  "ctf::$ARCHIVE//$COMPONENT:$VERSION" \
  "$TARGET" \
  --copy-resources
echo "Pushed: $TARGET"
echo

echo "=== Verify remote ==="
ocm get componentversions "$TARGET//$COMPONENT:$VERSION"

echo
echo "Component published:"
echo "  Registry: $TARGET"
echo "  Name:     $COMPONENT"
echo "  Version:  $VERSION"
echo "  Model ID: example-org/tiny-model"
echo
echo "Configure model-server to use it:"
echo "  ocm:"
echo "    repositories:"
echo "      - name: sample"
echo "        type: OCIRegistry"
echo "        url: $TARGET"
