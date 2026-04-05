#!/usr/bin/env python3
"""
Build a platform-specific wheel for run-on-business-day.

Usage: python build_wheel.py <version> <platform_tag> [wheel_build_dir]

Example:
  python build_wheel.py 2026.0.0 linux_x86_64
  python build_wheel.py 2026.0.0 linux_aarch64
  python build_wheel.py 2026.0.0 win_amd64
"""

import os
import sys
import pathlib
import hashlib
import zipfile
from datetime import datetime


def main():
    if len(sys.argv) < 3:
        print(__doc__)
        sys.exit(1)

    version = sys.argv[1]
    platform_tag = sys.argv[2]
    wheel_build_dir = sys.argv[3] if len(sys.argv) > 3 else f"wheel_build/{platform_tag}"

    build_wheel(version, platform_tag, wheel_build_dir)


def build_wheel(version, platform_tag, wheel_build_dir):
    wheel_build_path = pathlib.Path(wheel_build_dir)
    pkg_dir = wheel_build_path / "run_on_business_day"
    bin_dir = pkg_dir / "_bin"

    # Create directories if they don't exist
    bin_dir.mkdir(parents=True, exist_ok=True)

    # Copy Python files
    src_dir = pathlib.Path(__file__).parent / "run_on_business_day"
    for py_file in ["__init__.py", "_runner.py"]:
        src = src_dir / py_file
        dst = pkg_dir / py_file
        dst.write_text(src.read_text())

    # Binaries should already be in _bin_dir by CI
    # (This script assumes binaries are pre-placed at wheel_build_dir/run_on_business_day/_bin/)

    # Create .dist-info directory
    dist_info_name = f"run_on_business_day-{version}.dist-info"
    dist_info_dir = wheel_build_path / dist_info_name
    dist_info_dir.mkdir(parents=True, exist_ok=True)

    # Generate WHEEL file
    wheel_metadata = f"""Wheel-Version: 1.0
Generator: build_wheel.py
Root-Is-Purelib: false
Tag: py3-none-{platform_tag}
"""
    (dist_info_dir / "WHEEL").write_text(wheel_metadata)

    # Generate METADATA file (PyPI-compliant format)
    metadata = f"""Metadata-Version: 2.1
Name: run-on-business-day
Version: {version}
Summary: Run commands only on Japanese business days
Home-page: https://github.com/usaagi/run-on-business-day
Author: usaagi
License: MIT
Requires-Python: >=3.8
Classifier: License :: OSI Approved :: MIT License
Classifier: Operating System :: POSIX :: Linux
Classifier: Operating System :: Microsoft :: Windows
Classifier: Operating System :: MacOS
Classifier: Environment :: Console
Classifier: Programming Language :: Python :: 3
Classifier: Programming Language :: Python :: 3.8
Classifier: Topic :: Utilities
"""
    (dist_info_dir / "METADATA").write_text(metadata.strip() + "\n")

    # Generate entry_points.txt for console scripts
    entry_points = """[console_scripts]
run-on-business-day = run_on_business_day._runner:main
"""
    (dist_info_dir / "entry_points.txt").write_text(entry_points)

    # Create the wheel (zip file)
    dist_dir = pathlib.Path("dist")
    dist_dir.mkdir(exist_ok=True)

    wheel_filename = f"run_on_business_day-{version}-py3-none-{platform_tag}.whl"
    wheel_path = dist_dir / wheel_filename

    # Build file list and calculate hashes
    import base64
    records = []

    with zipfile.ZipFile(wheel_path, "w", zipfile.ZIP_DEFLATED) as whl:
        # Add package files
        for root, dirs, files in os.walk(wheel_build_path):
            for file in files:
                file_path = pathlib.Path(root) / file
                arcname = str(file_path.relative_to(wheel_build_path)).replace("\\", "/")

                with open(file_path, "rb") as f:
                    content = f.read()
                    hash_digest = hashlib.sha256(content).digest()
                    hash_b64 = base64.urlsafe_b64encode(hash_digest).decode().rstrip("=")
                    records.append(f"{arcname},sha256={hash_b64},{len(content)}")

                # ZipInfo を使ってパーミッション情報を保持
                zinfo = zipfile.ZipInfo(arcname)
                # _bin ディレクトリ内のバイナリは常に実行権限を付与
                if "_bin" in arcname:
                    zinfo.external_attr = (0o755 << 16)
                with open(file_path, "rb") as f:
                    whl.writestr(zinfo, f.read())

        # Add RECORD file (with no hash for itself)
        record_content = "\n".join(records) + f"\n{dist_info_name}/RECORD,,\n"
        whl.writestr(f"{dist_info_name}/RECORD", record_content)

    print(f"✓ Built wheel: {wheel_path}")
    return str(wheel_path)


if __name__ == "__main__":
    main()
