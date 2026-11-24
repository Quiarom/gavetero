#!/usr/bin/env bash
#
# install.sh - user-facing installer for Gavetero.
#
# Usage:
#   curl -sSf https://raw.githubusercontent.com/Quiarom/gavetero/main/install.sh | bash
#
# Environment variables (set ONE of these):
#   GAVETERO_VERSION=<tag>            - install that release tag
#   GAVETERO_COMMIT=<sha>             - install that commit (via GitHub Actions artifact)
#   (none)                           - install the latest stable release
#
# Behavior:
#   1. Detects the platform (prebuilt: linux/amd64 only).
#   2. Downloads the matching prebuilt binary.
#   3. Installs to ~/.local/bin/gavetero + symlink to gvt.
#   4. If ~/.local/bin is not on PATH, prints one line to add to the shell rc.
#   5. Verifies the install with `gvt version`.
#
# The script never requires sudo, never writes outside the user's
# home directory, and never modifies the system PATH. If anything
# fails, it prints a single actionable line and exits non-zero.

set -euo pipefail

REPO="Quiarom/gavetero"
BIN="gavetero"
DEST="${HOME}/.local/bin"

# ---------------------------------------------------------------------------
# Platform detection. We use uname -m for arch and uname -s for OS.
# ---------------------------------------------------------------------------
detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "${arch}" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "No prebuilt binary exists for architecture ${arch}." >&2; return 1 ;;
  esac
  case "${os}" in
    linux) os="linux" ;;
    darwin) os="darwin" ;;
    *) echo "No prebuilt binary exists for OS ${os}." >&2; return 1 ;;
  esac
  if [ "${os}_${arch}" != "linux_amd64" ]; then
    echo "No prebuilt binary exists for ${os}/${arch}." >&2
    return 1
  fi
  echo "${os}_${arch}"
}

verify_archive() {
  local archive="$1" checksums="$2" name="$3" expected actual
  expected="$(awk -v name="${name}" '$2 == name || $2 == "*" name { print $1 }' "${checksums}")"
  if [[ ! "${expected}" =~ ^[[:xdigit:]]{64}$ ]]; then
    echo "Checksum verification failed: missing or invalid entry for ${name}." >&2
    return 1
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "${archive}")" || return 1
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "${archive}")" || return 1
  else
    echo "Checksum verification requires sha256sum or shasum." >&2
    return 1
  fi
  actual="${actual%% *}"
  if [ "${actual,,}" != "${expected,,}" ]; then
    echo "Checksum verification failed: SHA-256 mismatch for ${name}." >&2
    return 1
  fi
  echo "Checksum OK: ${name}"
}

# ---------------------------------------------------------------------------
# Resolve the download URL.
#
# Priority:
#   1. GAVETERO_VERSION (a release tag) - GitHub Releases
#   2. GAVETERO_COMMIT (e.g. abc1234) - Actions artifact
#   3. neither - latest release (or fall back to make install-user)
# ---------------------------------------------------------------------------
resolve_url() {
  local platform="$1"
  if [ -n "${GAVETERO_VERSION:-}" ]; then
    echo "https://github.com/${REPO}/releases/download/${GAVETERO_VERSION}/${BIN}_${platform}.tar.gz"
  elif [ -n "${GAVETERO_COMMIT:-}" ]; then
    echo "https://github.com/${REPO}/actions/runs/${GAVETERO_COMMIT}/artifacts/${BIN}-${platform}"
  else
    echo "https://github.com/${REPO}/releases/latest/download/${BIN}_${platform}.tar.gz"
  fi
}

main() {
  echo "Gavetero installer"
  echo "=================="

  local platform
  if ! platform="$(detect_platform)"; then
    echo ""
    echo "This script does not support your platform. Build from source instead:"
    echo "  git clone https://github.com/${REPO}.git"
    echo "  cd gavetero"
    echo "  make install-user"
    exit 1
  fi
  echo ""
  echo "  platform : ${platform}"

  local url
  url="$(resolve_url "${platform}")"
  echo "  source   : ${url}"
  echo "  dest     : ${DEST}/${BIN}"
  echo ""

  # Download
  local tmpdir
  tmpdir="$(mktemp -d)"
  trap "rm -rf '${tmpdir}'" EXIT
  echo "Downloading..."
  if ! curl -fsSL -o "${tmpdir}/gavetero.tar.gz" "${url}"; then
    echo ""
    echo "Download failed."
    echo ""
    echo "If you are a developer working in the repo, use:"
    echo "  make install-user"
    echo ""
    echo "If you are a user, set GAVETERO_VERSION to a tag listed at https://github.com/${REPO}/releases:"
    echo "  curl -sSf https://raw.githubusercontent.com/${REPO}/main/install.sh | GAVETERO_VERSION=vX.Y.Z bash"
    exit 1
  fi

  if [ -n "${GAVETERO_VERSION:-}" ] || [ -z "${GAVETERO_COMMIT:-}" ]; then
    if ! curl -fsSL -o "${tmpdir}/checksums.txt" "${url%/*}/checksums.txt"; then
      echo "Checksum verification failed: could not download checksums.txt." >&2
      exit 1
    fi
    verify_archive "${tmpdir}/gavetero.tar.gz" "${tmpdir}/checksums.txt" "${BIN}_${platform}.tar.gz"
  fi

  # Extract (tarball layout: gavetero + gvt symlink target)
  tar -xzf "${tmpdir}/gavetero.tar.gz" -C "${tmpdir}"
  if [ ! -f "${tmpdir}/gavetero" ]; then
    echo ""
    echo "Downloaded archive does not contain 'gavetero'."
    exit 1
  fi

  # Install
  mkdir -p "${DEST}"
  install -m 0755 "${tmpdir}/gavetero" "${DEST}/gavetero"
  ln -sf "${BIN}" "${DEST}/gvt"

  # Verify
  echo ""
  echo "Installed:"
  echo "  ${DEST}/gavetero"
  echo "  ${DEST}/gvt -> gavetero"
  echo ""
  echo "Verify:"
  "${DEST}/gvt" version || true
  echo ""

  if echo "${PATH}" | tr ':' '\n' | grep -qx "${DEST}"; then
    echo "Ready. Try: gvt version"
  else
    echo "${DEST} is not on your PATH."
    echo "Add this line to ~/.bashrc (or ~/.zshrc, etc.):"
    echo "  export PATH=\"${DEST}:\${PATH}\""
    echo "Then open a new shell."
  fi
}

main "$@"
