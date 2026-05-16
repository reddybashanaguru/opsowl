#!/bin/bash
# Post-edit hook: auto-format based on file type
FILE="$1"
if [[ -z "$FILE" ]]; then exit 0; fi

if [[ "$FILE" == *.go ]]; then
  gofmt -w "$FILE" 2>/dev/null
  go vet ./... 2>/dev/null | head -20
elif [[ "$FILE" == *.ts || "$FILE" == *.tsx ]]; then
  cd apps/web && npx prettier --write "$FILE" 2>/dev/null
elif [[ "$FILE" == *.py ]]; then
  python -m black "$FILE" 2>/dev/null
fi
