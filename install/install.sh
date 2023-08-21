#/bin/bash

set -e

DIR_SCRIPT=$(dirname -- "${BASH_SOURCE[0]}")

if [[ "$(uname)" == "Darwin" ]]; then
    DIR_SCRIPT/install-mac.sh
else
    DIR_SCRIPT/install-linux.sh
fi
