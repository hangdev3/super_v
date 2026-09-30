#!/usr/bin/env bash
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
version=${SUPERV_VERSION:-0.1.0}
architecture=${SUPERV_ARCH:-$(dpkg --print-architecture)}
output_directory=${1:-"$project_root/build"}
package_name="super-v_${version}_${architecture}.deb"
output_path="$output_directory/$package_name"
staging_root=$(mktemp -d)

cleanup() {
	rm -rf "$staging_root"
}
trap cleanup EXIT

mkdir -p \
	"$output_directory" \
	"$staging_root/usr/bin" \
	"$staging_root/usr/share/applications" \
	"$staging_root/etc/xdg/autostart" \
	"$staging_root/DEBIAN"
chmod 0755 "$staging_root"

(cd "$project_root" && go build -buildvcs=false -o "$staging_root/usr/bin/super-v" ./cmd/super-v)
chmod 0755 "$staging_root/usr/bin/super-v"
install -m 0644 "$project_root/packaging/super-v.desktop" "$staging_root/usr/share/applications/super-v.desktop"
install -m 0644 "$project_root/packaging/super-v-autostart.desktop" "$staging_root/etc/xdg/autostart/super-v.desktop"
sed \
	-e "s/@VERSION@/$version/g" \
	-e "s/@ARCH@/$architecture/g" \
	"$project_root/packaging/debian/control.in" > "$staging_root/DEBIAN/control"

dpkg-deb --build --root-owner-group "$staging_root" "$output_path" >/dev/null
printf 'Pacote criado: %s\n' "$output_path"
