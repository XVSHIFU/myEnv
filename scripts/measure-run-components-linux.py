"""Run the gated Go component measurement from an isolated temporary copy.

This is a Linux diagnostic, not the full CLI/local-SSD performance gate.
The Go report records raw samples, binary digest, kernel, and filesystem types.
"""
import argparse
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("test_binary", type=Path)
parser.add_argument("output", type=Path)
args = parser.parse_args()
source = args.test_binary.resolve(strict=True)
output = args.output.resolve()
if output.exists():
    parser.error("output must be a new report path")
with tempfile.TemporaryDirectory(prefix="myenv-run-components-") as temporary:
    copied = Path(temporary) / "core.test"
    shutil.copyfile(source, copied)
    copied.chmod(0o700)
    env = dict(os.environ, MYENV_TEST_RUN_TIMINGS=str(output))
    subprocess.run([str(copied), "-test.run=^TestMeasureLinuxRunComponents$",
                    "-test.v", "-test.timeout=40s"],
                   env=env, check=True, timeout=50)
