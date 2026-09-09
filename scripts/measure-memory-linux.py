"""Record GNU time's per-command peak RSS; not simultaneous tree RSS."""
import argparse
import hashlib
import json
from pathlib import Path
import platform
import shutil
import subprocess
import tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("executable", type=Path)
parser.add_argument("output", type=Path)
args = parser.parse_args()
source = args.executable.resolve(strict=True)
with source.open("rb") as stream:
    digest = hashlib.file_digest(stream, "sha256").hexdigest()
records = []
with tempfile.TemporaryDirectory(prefix="myenv-rss-") as temporary:
    root = Path(temporary)
    executable = root / "myenv"
    shutil.copyfile(source, executable)
    executable.chmod(0o700)
    report = root / "time.txt"
    for flag in ("--version", "--help"):
        peaks = []
        for _ in range(10):
            subprocess.run(["/usr/bin/time", "-f", "%M", "-o", str(report),
                            str(executable), flag], check=True, timeout=10,
                           stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                           stderr=subprocess.PIPE)
            peaks.append(int(report.read_text().strip()))
        records.append(dict(flag=flag, platform=platform.platform(),
                            binary_sha256=digest, peak_rss_kib_samples=peaks,
                            max_peak_rss_kib=max(peaks),
                            method="GNU time %M on Linux; 10 separate informational commands; temporary executable copy; no runtime children"))
        print(flag, "max_peak_rss_kib", max(peaks))
args.output.parent.mkdir(parents=True, exist_ok=True)
args.output.write_text(json.dumps(records, indent=2) + "\n", encoding="utf-8")
