#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

check_application_security_opt() {
  file=$1
  count=0
  in_application=0
  in_security_opt=0
  cr=$(printf '\r')
  while IFS= read -r line || [ -n "$line" ]; do
    line=${line%"$cr"}
    case $line in
      "  sub2api:")
        in_application=1
        in_security_opt=0
        ;;
      "  "[A-Za-z0-9_-]*:)
        in_application=0
        in_security_opt=0
        ;;
      "    security_opt:")
        in_security_opt=$in_application
        ;;
      "      - no-new-privileges:true")
        if [ "$in_application" -eq 1 ] && [ "$in_security_opt" -eq 1 ]; then
          count=$((count + 1))
        fi
        ;;
      "    "[A-Za-z0-9_-]*:)
        in_security_opt=0
        ;;
    esac
  done < "$file"

  if [ "$count" -ne 1 ]; then
    printf '%s must enable no-new-privileges exactly once for the sub2api service\n' "$file" >&2
    return 1
  fi
}

run_scanner_fixture() {
  name=$1
  expected=$2
  fixture=$3
  if printf '%b' "$fixture" | check_application_security_opt /dev/stdin >/dev/null 2>&1; then
    actual=0
  else
    actual=1
  fi
  if [ "$actual" -ne "$expected" ]; then
    printf 'scanner fixture %s expected exit %s, got %s\n' "$name" "$expected" "$actual" >&2
    exit 1
  fi
}

run_scanner_fixture zero 1 '  sub2api:
    security_opt:
      - read-only'
run_scanner_fixture one 0 '  sub2api:
    security_opt:
      - no-new-privileges:true'
run_scanner_fixture adjacent-duplicate 1 '  sub2api:
    security_opt:
      - no-new-privileges:true
      - no-new-privileges:true'
run_scanner_fixture other-before 0 '  sub2api:
    security_opt:
      - read-only
      - no-new-privileges:true'
run_scanner_fixture separated-duplicate 1 '  sub2api:
    security_opt:
      - no-new-privileges:true
      - read-only
      - no-new-privileges:true'
run_scanner_fixture crlf 0 '  sub2api:\r\n    security_opt:\r\n      - no-new-privileges:true\r\n'

for compose_file in \
  deploy/docker-compose.yml \
  deploy/docker-compose.local.yml \
  deploy/docker-compose.standalone.yml \
  deploy/docker-compose.dev.yml
do
  check_application_security_opt "$compose_file"
done

printf 'docker compose security test passed\n'
