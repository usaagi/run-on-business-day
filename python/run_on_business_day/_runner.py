import os
import sys
import pathlib
import platform
import subprocess


def main():
    binary = pathlib.Path(__file__).parent / "_bin" / _binary_name()
    if not binary.exists():
        print(f"Error: binary not found at {binary}", file=sys.stderr)
        sys.exit(1)

    if platform.system() == "Windows":
        # Windows doesn't support os.execv, use subprocess instead
        sys.exit(subprocess.run([str(binary)] + sys.argv[1:]).returncode)
    else:
        # Unix-like systems: replace current process with binary
        os.execv(str(binary), [str(binary)] + sys.argv[1:])


def _binary_name():
    s, m = platform.system(), platform.machine()
    if s == "Linux" and m == "x86_64":
        return "run-on-business-day"
    if s == "Linux" and m == "aarch64":
        return "run-on-business-day-arm64"
    if s == "Windows":
        return "run-on-business-day.exe"
    raise RuntimeError(f"Unsupported platform: {s}/{m}")


if __name__ == "__main__":
    main()
