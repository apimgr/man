#!/bin/bash
# scripts/verify-licenses.sh
#
# License compliance check for casman dependencies.
# Fails build if any GPL/AGPL/LGPL dependency is detected.
# See AI.md PART 2 for license policy.

set -e

echo "Checking for incompatible licenses..."

if ! command -v go-licenses &> /dev/null; then
    echo "Installing go-licenses..."
    go install github.com/google/go-licenses@latest
fi

echo "Scanning dependencies..."
if go-licenses csv ./... | grep -iE 'GPL|AGPL|LGPL'; then
    echo "ERROR: Copyleft license detected!"
    echo "Remove the dependency or find an alternative."
    exit 1
fi

echo "All licenses are compatible"

echo "Generating license report..."
go-licenses csv ./... > licenses.csv
go-licenses save ./... --save_path=third_party_licenses

echo "License report saved to licenses.csv and third_party_licenses/"
echo ""
echo "Next steps:"
echo "1. Review licenses.csv"
echo "2. Update LICENSE.md with any new dependencies"
echo "3. Commit the changes"
