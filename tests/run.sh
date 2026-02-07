#!/usr/bin/env bash

set -euo pipefail

bash /workspace/tests/test_profiles.sh
bash /workspace/test_squid_gen.sh

echo "All tests passed."
