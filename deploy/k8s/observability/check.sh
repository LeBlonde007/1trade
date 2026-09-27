#!/usr/bin/env bash
# Offline verification of the observability config (`make obs-check`):
#   1. every PromQL expression in alerts.yaml and dashboards/*.json uses only metrics the Go services
#      actually declare (so a renamed metric can't leave an alert or a panel silently empty);
#   2. promtool check + unit tests of the alert rules (alerts.test.yaml).
# Needs python3 + PyYAML and promtool on PATH (or PROMTOOL=/path/to/promtool).
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"; ROOT="$(cd "$HERE/../../.." && pwd)"
PROMTOOL="${PROMTOOL:-promtool}"
python3 - "$HERE" "$ROOT" <<'PY'
import glob, json, re, sys, yaml
here, root = sys.argv[1], sys.argv[2]
# Metric names the services declare (promauto Name: "...") plus the shared obs package + Prometheus'.
declared = set()
for f in glob.glob(f"{root}/services/*/internal/**/*.go", recursive=True):
    if f.endswith("_test.go"): continue
    declared |= set(re.findall(r'Name:\s+"([a-z_][a-z0-9_]*)"', open(f).read()))
declared |= {"up"}
hist = {m for m in declared if m.endswith("_seconds")}
declared |= {m + s for m in hist for s in ("_bucket", "_sum", "_count")}
spec = yaml.safe_load(open(f"{here}/alerts.yaml"))["spec"]
yaml.safe_dump({"groups": spec["groups"]}, open(f"{here}/rules.generated.yaml", "w"), sort_keys=False)
exprs = [(f"alert {r['alert']}", r["expr"]) for g in spec["groups"] for r in g["rules"]]
for d in glob.glob(f"{here}/dashboards/*.json"):
    for p in json.load(open(d))["panels"]:
        for t in p.get("targets", []):
            exprs.append((f"{d.rsplit('/',1)[1]}: {p['title']}", t["expr"]))
funcs = {"sum","rate","increase","histogram_quantile","by","and","or","vector","without","max","min","avg","count","topk","le","clamp_min"}
bad = []
for where, e in exprs:
    e2 = re.sub(r'\{[^}]*\}', '', e)           # drop label matchers
    e2 = re.sub(r'\[[^\]]*\]', '', e2)          # drop ranges
    e2 = re.sub(r'by\s*\([^)]*\)', '', e2)      # drop grouping labels
    for tok in re.findall(r'[a-z_][a-z0-9_]*', e2):
        if tok in funcs: continue
        if tok not in declared: bad.append(f"{where}: unknown metric {tok!r}")
if bad:
    print("\n".join(bad)); sys.exit(1)
print(f"metric names: {len(exprs)} expressions reference only declared metrics ({len(declared)} known)")
PY
cd "$HERE"
"$PROMTOOL" check rules rules.generated.yaml
"$PROMTOOL" test rules alerts.test.yaml
