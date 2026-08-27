#!/usr/bin/env bash
set -eu -o pipefail

# Renders the Traefik Gateway API operator install manifest used by the operator
# conformance suite, from a checkout of github.com/traefik/operator.
#
# Usage: script/gateway-api-operator-fixture.sh <path to the operator checkout>

operator_dir="${1:-../operator}"
fixture="integration/fixtures/gateway-api-conformance/01-operator.yml"

for file in config/crd/gateway.traefik.io_traefikproxies.yaml \
            config/rbac/role.yaml \
            config/rbac/dataplane_role.yaml \
            config/manager/manager.yaml; do
    if [ ! -f "${operator_dir}/${file}" ]; then
        echo "Missing ${operator_dir}/${file}: is ${operator_dir} a traefik/operator checkout?" >&2
        exit 1
    fi
done

{
    echo "# Traefik Gateway API operator install."
    echo "#"
    echo "# Rendered from the operator repository (config/crd, config/rbac, config/manager)."
    echo "# Regenerate with: make generate-gateway-api-operator-fixture"
    for file in config/crd/gateway.traefik.io_traefikproxies.yaml \
                config/rbac/role.yaml \
                config/rbac/dataplane_role.yaml \
                config/manager/manager.yaml; do
        echo "---"
        cat "${operator_dir}/${file}"
    done
} > "${fixture}"

# The operator image is side loaded into the k3s node, never pulled, and leader
# election only delays the startup of a single replica.
python3 - "${fixture}" <<'EOF'
import io
import sys

path = sys.argv[1]
content = io.open(path, encoding="utf-8").read()

replacements = [
    ("        image: traefik/operator:latest\n        args:",
     "        image: traefik/operator:latest\n        imagePullPolicy: Never\n        args:"),
    ("        args:\n        - --leader-elect\n        - --health-probe-bind-address=:8081",
     "        args:\n        - --health-probe-bind-address=:8081"),
]

for old, new in replacements:
    if content.count(old) != 1:
        sys.exit("Unexpected manager manifest, cannot apply: %s" % old)
    content = content.replace(old, new)

io.open(path, "w", encoding="utf-8").write(content)
EOF

echo "Rendered ${fixture}"
