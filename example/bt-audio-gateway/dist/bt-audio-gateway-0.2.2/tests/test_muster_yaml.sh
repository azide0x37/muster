#!/bin/sh
set -eu

test -f muster.yaml

for required in \
  "framework: Muster" \
  "name: bt-audio-gateway" \
  "manager: systemd" \
  "update_polling: systemd-timer" \
  "required_tool: uv"
do
  grep -q "$required" muster.yaml
done

awk '
function finish_component() {
  if (!in_component) {
    return
  }
  if (!has_what || !has_why) {
    printf "component %s is missing non-empty literate.what or literate.why\n", component_id > "/dev/stderr"
    failures++
  }
}

$0 == "  components:" {
  in_components = 1
  next
}

in_components && $0 == "  edges:" {
  finish_component()
  in_component = 0
  in_components = 0
  next
}

in_components && /^    - id: / {
  finish_component()
  component_id = $0
  sub(/^    - id: /, "", component_id)
  in_component = 1
  has_what = 0
  has_why = 0
  next
}

in_component && /^        what:[[:space:]]*[^[:space:]]/ { has_what = 1 }
in_component && /^        why:[[:space:]]*[^[:space:]]/ { has_why = 1 }

END {
  if (in_components) {
    finish_component()
  }
  exit failures != 0
}
' muster.yaml
