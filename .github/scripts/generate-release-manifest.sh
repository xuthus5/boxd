#!/usr/bin/env bash
set -euo pipefail

assets_dir="${1:?assets directory is required}"
repository="${2:?repository is required}"
release_tag="${3:?release tag is required}"
version="${4:?version is required}"
commit="${5:?commit is required}"
published_at="${6:?published time is required}"

assets_dir="$(cd "${assets_dir}" && pwd)"
base_url="https://github.com/${repository}/releases/download/${release_tag}"
container_image="ghcr.io/${repository,,}"

asset_json() {
  local name="$1"
  local file="${assets_dir}/${name}"

  if [[ ! -f "${file}" ]]; then
    echo "missing release asset: ${file}" >&2
    exit 1
  fi

  local size sha256
  size="$(stat -c '%s' "${file}")"
  sha256="$(sha256sum "${file}" | awk '{print $1}')"

  jq -cn \
    --arg name "${name}" \
    --arg url "${base_url}/${name}" \
    --argjson size "${size}" \
    --arg sha256 "${sha256}" \
    '{name: $name, url: $url, size: $size, sha256: $sha256}'
}

amd64_binary="$(asset_json boxd-linux-amd64)"
amd64_archive="$(asset_json boxd-linux-amd64.tar.gz)"
arm64_binary="$(asset_json boxd-linux-arm64)"
arm64_archive="$(asset_json boxd-linux-arm64.tar.gz)"

jq -n \
  --arg version "${version}" \
  --arg release_tag "${release_tag}" \
  --arg repository "${repository}" \
  --arg commit "${commit}" \
  --arg published_at "${published_at}" \
  --arg container_image "${container_image}" \
  --argjson amd64_binary "${amd64_binary}" \
  --argjson amd64_archive "${amd64_archive}" \
  --argjson arm64_binary "${arm64_binary}" \
  --argjson arm64_archive "${arm64_archive}" \
  '{
    schemaVersion: 1,
    version: $version,
    releaseTag: $release_tag,
    repository: $repository,
    commit: $commit,
    publishedAt: $published_at,
    container: {
      registry: "ghcr.io",
      image: $container_image,
      tag: $release_tag
    },
    assets: {
      linux: {
        amd64: {binary: $amd64_binary, archive: $amd64_archive},
        arm64: {binary: $arm64_binary, archive: $arm64_archive}
      }
    }
  }' > "${assets_dir}/boxd-manifest.json"

jq -e '
  .schemaVersion == 1
  and (.container.image | startswith("ghcr.io/"))
  and (.container.tag | length > 0)
  and (.assets.linux.amd64.binary.sha256 | length == 64)
  and (.assets.linux.amd64.archive.sha256 | length == 64)
  and (.assets.linux.arm64.binary.sha256 | length == 64)
  and (.assets.linux.arm64.archive.sha256 | length == 64)
' "${assets_dir}/boxd-manifest.json" >/dev/null
