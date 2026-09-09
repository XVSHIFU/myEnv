"""Measure informational commands on a Unix host without shell startup."""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import platform
import shutil
import subprocess
import tempfile
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("executable", type=Path)
parser.add_argument("output", type=Path)
parser.add_argument("--samples", type=int, default=50)
args = parser.parse_args()
if not 20 <= args.samples <= 1000:
    parser.error("samples must be between 20 and 1000")
source = args.executable.resolve(strict=True)
with source.open("rb") as stream:
    digest = hashlib.file_digest(stream, "sha256").hexdigest()
results = []
with tempfile.TemporaryDirectory(prefix="myenv-startup-") as temporary:
    copied = Path(temporary) / "myenv"
    shutil.copyfile(source, copied)
    copied.chmod(0o700)
    for location, executable in (("original", source), ("temporary_copy", copied)):
        for flag in ("--version", "--help"):
            samples = []
            for _ in range(args.samples + 1):
                start = time.perf_counter_ns()
                subprocess.run([str(executable), flag], check=True, timeout=10,
                               stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                               stderr=subprocess.PIPE)
                samples.append((time.perf_counter_ns() - start) / 1_000_000)
            ordered = sorted(samples[1:])
            result = dict(location=location, executable=str(executable), flag=flag,
                          binary_sha256=digest, platform=platform.platform(),
                          first_observed_ms=samples[0], samples_ms=samples[1:],
                          median_ms=ordered[math.ceil(args.samples * .5) - 1],
                          p95_ms=ordered[math.ceil(args.samples * .95) - 1],
                          filesystem_device=os.stat(executable).st_dev,
                          method="subprocess, no shell; stdout discarded; first sample excluded; no cache eviction")
            results.append(result)
            print(location, flag, "median_ms", result["median_ms"], "p95_ms", result["p95_ms"])
args.output.parent.mkdir(parents=True, exist_ok=True)
args.output.write_text(json.dumps(results, indent=2) + "\n", encoding="utf-8")
