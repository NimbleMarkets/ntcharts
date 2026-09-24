#!/usr/bin/env python3
"""Summarize raw probe JSON; no third-party dependencies."""
import argparse
import json
import math
from pathlib import Path
import statistics

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('directory', type=Path, help='directory containing probe JSON reports')
root = parser.parse_args().directory
print('| Report | Replies / probes | Median ms | P95 ms | Max ms |')
print('| --- | ---: | ---: | ---: | ---: |')
for path in sorted(root.glob('*.json')):
    report = json.loads(path.read_text())
    rows = report.get('results') or []
    values = sorted(row['latency_ms'] for row in rows if 'latency_ms' in row)
    if values:
        median = statistics.median(values)
        p95 = values[math.ceil(0.95 * len(values)) - 1]
        cells = f'{median:.3f} | {p95:.3f} | {values[-1]:.3f}'
    else:
        cells = '— | — | —'
    print(f'| {path.name} | {len(values)} / {len(rows)} | {cells} |')
