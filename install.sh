#!/usr/bin/env bash
set -euo pipefail

REPO="${SAGE_INSTALL_REPO:-divijg19/sage}"
BASE_URL="${SAGE_INSTALL_BASE_URL:-https://github.com/${REPO}/releases/download}"
LATEST_API_URL="${SAGE_INSTALL_LATEST_API_URL:-https://api.github.com/repos/${REPO}/releases/latest}"
BIN_DIR="${HOME}/.local/bin"
VERSION=""
ENABLE_ALIAS=0
TARGET_SHELL=""

usage() {
  cat <<'EOF'
Install Sage from a GitHub release.

Usage:
  ./install.sh [--version vX.Y.Z] [--bin-dir DIR] [--alias] [--shell bash|zsh]

Options:
  --version TAG              Install a specific release tag. Defaults to the latest release.
  --bin-dir DIR              Install the sage binary into DIR. Defaults to ~/.local/bin.
  --alias                    Append an opt-in shell function so `chronicle` runs `sage tui`.
  --shell bash|zsh           Choose which shell rc file to update when aliasing is enabled.
  -h, --help                 Show this help text.
EOF
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)
      echo "unsupported operating system: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

resolve_version() {
  if [[ -n "${VERSION}" ]]; then
    echo "${VERSION}"
    return
  fi

  if [[ -n "${SAGE_INSTALL_LATEST_TAG:-}" ]]; then
    echo "${SAGE_INSTALL_LATEST_TAG}"
    return
  fi

  local tag
  tag="$(curl -fsSL "${LATEST_API_URL}" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  if [[ -z "${tag}" ]]; then
    echo "failed to resolve the latest Sage release tag" >&2
    exit 1
  fi
  echo "${tag}"
}

asset_name_for() {
  local version="$1"
  local os_name="$2"
  local arch="$3"
  echo "sage_${version}_${os_name}_${arch}.tar.gz"
}

asset_url_for() {
  local version="$1"
  local os_name="$2"
  local arch="$3"
  local asset_name
  asset_name="$(asset_name_for "${version}" "${os_name}" "${arch}")"
  echo "${BASE_URL}/${version}/${asset_name}"
}

resolve_rc_file() {
  local shell_name="$1"
  if [[ -n "${SAGE_INSTALL_RC_FILE:-}" ]]; then
    echo "${SAGE_INSTALL_RC_FILE}"
    return
  fi

  case "${shell_name}" in
    zsh) echo "${HOME}/.zshrc" ;;
    *) echo "${HOME}/.bashrc" ;;
  esac
}

chronicle_alias_block() {
  cat <<'EOF'
# >>> sage chronicle alias >>>
chronicle() {
  sage tui "$@"
}
# <<< sage chronicle alias <<<
EOF
}

append_chronicle_alias_block() {
  local rc_file="$1"
  mkdir -p "$(dirname "${rc_file}")"
  touch "${rc_file}"

  if grep -Fq '# >>> sage chronicle alias >>>' "${rc_file}"; then
    return
  fi

  {
    printf '\n'
    chronicle_alias_block
    printf '\n'
  } >> "${rc_file}"
}

install_binary() {
  local source_bin="$1"
  mkdir -p "${BIN_DIR}"
  if command -v install >/dev/null 2>&1; then
    install -m 0755 "${source_bin}" "${BIN_DIR}/sage"
  else
    cp "${source_bin}" "${BIN_DIR}/sage"
    chmod 0755 "${BIN_DIR}/sage"
  fi
}

path_hint() {
  case ":${PATH}:" in
    *":${BIN_DIR}:"*) return 0 ;;
  esac

  echo "Add ${BIN_DIR} to your PATH:"
  echo "  export PATH=\"${BIN_DIR}:\$PATH\""
}

main() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --version)
        VERSION="${2:-}"
        shift 2
        ;;
      --bin-dir)
        BIN_DIR="${2:-}"
        shift 2
        ;;
      --alias)
        ENABLE_ALIAS=1
        shift
        ;;
      --shell)
        TARGET_SHELL="${2:-}"
        shift 2
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        echo "unknown argument: $1" >&2
        usage >&2
        exit 1
        ;;
    esac
  done

  local os_name arch version asset_url asset_name tmp_dir archive_path extract_dir rc_file
  os_name="$(detect_os)"
  arch="$(detect_arch)"
  version="$(resolve_version)"
  asset_name="$(asset_name_for "${version}" "${os_name}" "${arch}")"
  asset_url="$(asset_url_for "${version}" "${os_name}" "${arch}")"
  tmp_dir="$(mktemp -d)"
  archive_path="${tmp_dir}/${asset_name}"
  extract_dir="${tmp_dir}/extract"
  trap 'rm -rf "${tmp_dir:-}"' EXIT

  echo "Installing Sage ${version} for ${os_name}/${arch}..."
  curl -fsSL "${asset_url}" -o "${archive_path}"
  mkdir -p "${extract_dir}"
  tar -xzf "${archive_path}" -C "${extract_dir}"

  if [[ ! -f "${extract_dir}/sage" ]]; then
    echo "release archive did not contain a sage binary" >&2
    exit 1
  fi

  install_binary "${extract_dir}/sage"
  echo "Installed: ${BIN_DIR}/sage"

  if [[ "${ENABLE_ALIAS}" -eq 1 ]]; then
    if [[ -z "${TARGET_SHELL}" ]]; then
      TARGET_SHELL="bash"
    fi
    rc_file="$(resolve_rc_file "${TARGET_SHELL}")"
    append_chronicle_alias_block "${rc_file}"
    echo "Added Chronicle alias to ${rc_file}"
  fi

  path_hint || true
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
