#!/usr/bin/env bash
set -euo pipefail

package_path=${1:?uso: test-deb.sh caminho/do/pacote.deb}

[[ -f "$package_path" ]]
[[ $(dpkg-deb --field "$package_path" Package) == "super-v" ]]
[[ $(dpkg-deb --field "$package_path" Architecture) == "$(dpkg --print-architecture)" ]]

contents=$(dpkg-deb --contents "$package_path")
for expected_path in \
	"./usr/bin/super-v" \
	"./usr/share/applications/super-v.desktop" \
	"./etc/xdg/autostart/super-v.desktop"; do
	if ! printf '%s\n' "$contents" | rg -q " $expected_path$"; then
		printf 'arquivo ausente no pacote: %s\n' "$expected_path" >&2
		exit 1
	fi
done

if ! printf '%s\n' "$contents" | rg -q -- '-rwxr-xr-x .* ./usr/bin/super-v$'; then
	printf 'executavel sem permissao correta no pacote\n' >&2
	exit 1
fi

printf 'Smoke test do pacote aprovado: %s\n' "$package_path"
