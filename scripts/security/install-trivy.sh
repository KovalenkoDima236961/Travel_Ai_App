#!/usr/bin/env bash
set -euo pipefail

version="${TRIVY_VERSION:-0.70.0}"
version="${version#v}"
install_dir="${TRIVY_INSTALL_DIR:-${HOME}/.local/bin}"

if [[ "${version}" != "0.70.0" ]]; then
  echo "Unsupported Trivy version ${version}; update install-trivy.sh checksums first." >&2
  exit 2
fi

case "$(uname -s)-$(uname -m)" in
  Linux-x86_64)
    asset="trivy_${version}_Linux-64bit.tar.gz"
    sha256="8b4376d5d6befe5c24d503f10ff136d9e0c49f9127a4279fd110b727929a5aa9"
    ;;
  Linux-aarch64|Linux-arm64)
    asset="trivy_${version}_Linux-ARM64.tar.gz"
    sha256="2f6bb988b553a1bbac6bdd1ce890f5e412439564e17522b88a4541b4f364fc8d"
    ;;
  Darwin-x86_64)
    asset="trivy_${version}_macOS-64bit.tar.gz"
    sha256="52d531452b19e7593da29366007d02a810e1e0080d02f9cf6a1afb46c35aaa93"
    ;;
  Darwin-arm64)
    asset="trivy_${version}_macOS-ARM64.tar.gz"
    sha256="68e543c51dcc96e1c344053a4fde9660cf602c25565d9f09dc17dd41e13b838a"
    ;;
  *)
    echo "Unsupported platform for Trivy install: $(uname -s)-$(uname -m)" >&2
    exit 2
    ;;
esac

url="https://github.com/aquasecurity/trivy/releases/download/v${version}/${asset}"
tmpdir="$(mktemp -d)"
archive="${tmpdir}/${asset}"
trap 'rm -rf "${tmpdir}"' EXIT

mkdir -p "${install_dir}"
curl -fsSL --retry 3 --retry-delay 2 -o "${archive}" "${url}"

if command -v sha256sum >/dev/null 2>&1; then
  printf '%s  %s\n' "${sha256}" "${archive}" | sha256sum -c -
else
  actual="$(shasum -a 256 "${archive}" | awk '{print $1}')"
  if [[ "${actual}" != "${sha256}" ]]; then
    echo "Checksum mismatch for ${asset}: expected ${sha256}, got ${actual}" >&2
    exit 1
  fi
fi

tar -xzf "${archive}" -C "${tmpdir}" trivy
install -m 0755 "${tmpdir}/trivy" "${install_dir}/trivy"

if [[ -n "${GITHUB_PATH:-}" ]]; then
  echo "${install_dir}" >> "${GITHUB_PATH}"
fi

"${install_dir}/trivy" --version
