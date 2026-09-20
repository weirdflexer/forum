#!/usr/bin/env python3
"""Compare registered API method/path pairs with the committed OpenAPI contract."""
import json
import re
from pathlib import Path
root = Path(__file__).resolve().parent.parent
source = (root / 'backend/internal/forum/server.go').read_text()
actual = {(method.lower(), path.removeprefix('/api/v1')) for method, path in re.findall(r'"(GET|POST|PATCH|DELETE) (/api/v1[^" ]+)"', source)}
contract = json.loads((root / 'docs/openapi.json').read_text())
expected = {(method, path) for path, methods in contract['paths'].items() for method in methods}
if actual != expected:
    raise SystemExit(f'OpenAPI mismatch: undocumented={actual-expected}, unimplemented={expected-actual}')
print(f'OpenAPI: {len(actual)} routes match the server.')
