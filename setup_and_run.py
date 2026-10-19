#!/usr/bin/env python3
"""
setup_and_run.py - one-command setup + launcher for Videdit (reel-generator, web-ui branch).

Run it from anywhere inside a clone of the repo:

    python setup_and_run.py              # check, install what is missing, start the web UI
    python setup_and_run.py --check      # only report what is installed / missing
    python setup_and_run.py --cpu        # install the smaller CPU-only PyTorch (no NVIDIA GPU)
    python setup_and_run.py -- "videos/clip.mp4"      # extra args are passed to cmd/editor

Everything it installs lives in <repo>/.videdit-env/ (delete that folder to undo it):
    venv/   Python virtual environment: openai-whisper, yt-dlp, indic-transliteration
    go/     Go toolchain (only downloaded if your Go is missing or older than go.mod needs)
    shim/   small `whisper` wrapper that falls back to CPU when CUDA is unavailable
    (ffmpeg) a static ffmpeg/ffprobe via the `static-ffmpeg` pip package, only if yours is
             missing or lacks libx264 / the `ass` subtitle filter

Nothing is installed system-wide, and no admin rights are needed.
The only thing it cannot install for you is Python 3.10+ itself (needed to run this file).
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import platform
import re
import shutil
import socket
import subprocess
import sys
import tarfile
import threading
import time
import urllib.request
import venv
import webbrowser
import zipfile
from pathlib import Path

# ----------------------------------------------------------------------------- config
GO_VERSION = "go1.25.1"          # must satisfy go.mod ("go 1.25.1")
MIN_GO = (1, 25, 1)
MIN_PY = (3, 10)
PORT = 8080                      # hardcoded in cmd/editor (srv.Start(8080))
URL = f"http://localhost:{PORT}"
CPU_TORCH_INDEX = "https://download.pytorch.org/whl/cpu"

REPO = Path(__file__).resolve().parent
ENV = REPO / ".videdit-env"
VENV = ENV / "venv"
GO_DIR = ENV / "go"
SHIM = ENV / "shim"
FFMPEG_MARKER = ENV / "ffmpeg_dir.txt"

IS_WIN = os.name == "nt"
BIN = "Scripts" if IS_WIN else "bin"
EXE = ".exe" if IS_WIN else ""
VPY = VENV / BIN / f"python{EXE}"
LOCAL_GO = GO_DIR / "bin" / f"go{EXE}"

problems: list[str] = []
todo: list[str] = []      # things setup will install (reported by --check)


# ----------------------------------------------------------------------------- output
def step(msg: str) -> None:
    print(f"\n==> {msg}", flush=True)


def ok(name: str, detail: str = "") -> None:
    print(f"  [ ok ] {name:<24} {detail}", flush=True)


def warn(name: str, detail: str = "") -> None:
    print(f"  [warn] {name:<24} {detail}", flush=True)


def bad(name: str, detail: str = "") -> None:
    print(f"  [FAIL] {name:<24} {detail}", flush=True)
    problems.append(name)


def die(msg: str, code: int = 1) -> None:
    print(f"\nERROR: {msg}", file=sys.stderr, flush=True)
    sys.exit(code)


# ----------------------------------------------------------------------------- helpers
def capture(cmd: list, timeout: int = 60):
    """Run a command, return (returncode, combined output) or None if it cannot start."""
    try:
        p = subprocess.run([str(c) for c in cmd], capture_output=True, text=True,
                           errors="replace", timeout=timeout)
        return p.returncode, (p.stdout or "") + (p.stderr or "")
    except (OSError, subprocess.SubprocessError):
        return None


def run_visible(cmd: list, **kw) -> int:
    """Run a command with its output streaming to the terminal."""
    return subprocess.call([str(c) for c in cmd], **kw)


def parse_go_version(text: str):
    m = re.search(r"go(\d+)\.(\d+)(?:\.(\d+))?", text or "")
    return tuple(int(x or 0) for x in m.groups()) if m else None


def port_in_use(port: int) -> bool:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.settimeout(0.5)
        return s.connect_ex(("127.0.0.1", port)) == 0


def download(url: str, dest: Path, sha256: str | None = None) -> None:
    req = urllib.request.Request(url, headers={"User-Agent": "videdit-setup"})
    h = hashlib.sha256()
    try:
        with urllib.request.urlopen(req, timeout=60) as r, open(dest, "wb") as f:
            total = int(r.headers.get("Content-Length") or 0)
            done, last = 0, -1
            while True:
                chunk = r.read(1 << 20)
                if not chunk:
                    break
                f.write(chunk)
                h.update(chunk)
                done += len(chunk)
                if total:
                    pct = done * 100 // total
                    if pct != last and pct % 10 == 0:
                        print(f"      {pct}% of {total // (1 << 20)} MB", flush=True)
                        last = pct
    except Exception as e:  # noqa: BLE001
        hint = ""
        if "CERTIFICATE_VERIFY_FAILED" in str(e) and platform.system() == "Darwin":
            hint = " (macOS python.org build: run 'Install Certificates.command' once)"
        raise RuntimeError(f"download failed for {url}: {e}{hint}") from e
    if sha256 and h.hexdigest() != sha256:
        raise RuntimeError(f"checksum mismatch for {url}")


def safe_extract_tar(archive: Path, dest: Path) -> None:
    with tarfile.open(archive) as t:
        root = dest.resolve()
        for m in t.getmembers():
            if not (dest / m.name).resolve().is_relative_to(root):
                raise RuntimeError(f"unsafe path in archive: {m.name}")
        t.extractall(dest)


# ----------------------------------------------------------------------------- Go
def probe_go():
    """Return (path, version_tuple) of a usable Go (>= go.mod's version), else None."""
    for cand in (str(LOCAL_GO), shutil.which("go")):
        if not cand or not Path(cand).exists():
            continue
        r = capture([cand, "version"], timeout=30)
        if r and r[0] == 0:
            v = parse_go_version(r[1])
            # A system Go older than go.mod would try to auto-download a toolchain; we
            # only accept a Go that is new enough on its own.
            if v and v >= MIN_GO:
                return cand, v
    return None


def install_go() -> None:
    step(f"Downloading {GO_VERSION} into {GO_DIR.relative_to(REPO)} (about 70 MB)")
    osname = {"Windows": "windows", "Darwin": "darwin", "Linux": "linux"}.get(platform.system())
    arch = {"x86_64": "amd64", "amd64": "amd64", "arm64": "arm64", "aarch64": "arm64",
            "i386": "386", "i686": "386", "armv7l": "armv6l"}.get(platform.machine().lower())
    if not osname or not arch:
        die(f"Unsupported platform {platform.system()}/{platform.machine()} for the Go download. "
            f"Install Go {GO_VERSION[2:]}+ from https://go.dev/dl/ and re-run.")
    try:
        req = urllib.request.Request("https://go.dev/dl/?mode=json&include=all",
                                     headers={"User-Agent": "videdit-setup"})
        with urllib.request.urlopen(req, timeout=60) as r:
            releases = json.load(r)
        rel = next(x for x in releases if x["version"] == GO_VERSION)
        f = next(x for x in rel["files"]
                 if x["os"] == osname and x["arch"] == arch and x["kind"] == "archive")
    except Exception as e:  # noqa: BLE001
        die(f"Could not look up the Go download ({e}). Install Go {GO_VERSION[2:]}+ from "
            "https://go.dev/dl/ and re-run.")
    ENV.mkdir(exist_ok=True)
    archive = ENV / f["filename"]
    try:
        download("https://go.dev/dl/" + f["filename"], archive, f["sha256"])
        tmp = ENV / "_go_extract"
        shutil.rmtree(tmp, ignore_errors=True)
        tmp.mkdir()
        if archive.suffix == ".zip":
            with zipfile.ZipFile(archive) as z:
                z.extractall(tmp)
        else:
            safe_extract_tar(archive, tmp)
        shutil.rmtree(GO_DIR, ignore_errors=True)
        shutil.move(str(tmp / "go"), str(GO_DIR))
        shutil.rmtree(tmp, ignore_errors=True)
    except Exception as e:  # noqa: BLE001
        die(str(e))
    finally:
        archive.unlink(missing_ok=True)


# ----------------------------------------------------------------------------- ffmpeg
def ffmpeg_capable(ffmpeg: str) -> tuple[bool, str]:
    enc = capture([ffmpeg, "-hide_banner", "-encoders"])
    fil = capture([ffmpeg, "-hide_banner", "-filters"])
    if not enc or not fil:
        return False, "cannot run"
    missing = []
    if "libx264" not in enc[1]:
        missing.append("libx264")
    if not re.search(r"^\s*\S+\s+ass\s", fil[1], re.M):
        missing.append("'ass' filter (libass)")
    if not re.search(r"\baac\b", enc[1]):
        missing.append("aac")
    return (not missing), ("missing " + ", ".join(missing)) if missing else ""


def probe_ffmpeg():
    """Return (ffmpeg_dir_or_None, description) for a usable ffmpeg+ffprobe, else None."""
    sys_ff, sys_fp = shutil.which("ffmpeg"), shutil.which("ffprobe")
    if sys_ff and sys_fp:
        good, why = ffmpeg_capable(sys_ff)
        if good:
            return None, f"system ({sys_ff})"
        warn("ffmpeg (system)", why)
    if FFMPEG_MARKER.exists():
        d = Path(FFMPEG_MARKER.read_text().strip())
        if (d / f"ffmpeg{EXE}").exists() and (d / f"ffprobe{EXE}").exists():
            return d, f"bundled static build ({d})"
    return None


def ensure_ffmpeg() -> Path | None:
    found = probe_ffmpeg()
    if found:
        return found[0]
    step("ffmpeg/ffprobe missing or incomplete -> installing a static build (static-ffmpeg)")
    pip_install(["static-ffmpeg"])
    snippet = ("import shutil, static_ffmpeg; static_ffmpeg.add_paths(); "
               "print(shutil.which('ffmpeg')); print(shutil.which('ffprobe'))")
    r = capture([VPY, "-c", snippet], timeout=900)
    lines = [ln.strip() for ln in (r[1] if r else "").splitlines() if ln.strip()]
    if not r or r[0] != 0 or len(lines) < 2 or "None" in lines[-2:]:
        die("Could not obtain a static ffmpeg. Install ffmpeg yourself "
            "(Windows: winget install Gyan.FFmpeg | macOS: brew install ffmpeg | "
            "Linux: sudo apt install ffmpeg) and re-run.\n" + (r[1][-600:] if r else ""))
    ffmpeg_path = Path(lines[-2])
    good, why = ffmpeg_capable(str(ffmpeg_path))
    if not good:
        die(f"The bundled ffmpeg is unusable ({why}). Install a full ffmpeg build and re-run.")
    FFMPEG_MARKER.write_text(str(ffmpeg_path.parent))
    return ffmpeg_path.parent


# ----------------------------------------------------------------------------- Python env
def ensure_venv() -> None:
    if VPY.exists():
        return
    step(f"Creating Python virtual environment in {VENV.relative_to(REPO)}")
    try:
        ENV.mkdir(exist_ok=True)
        venv.create(VENV, with_pip=True, clear=False)
    except Exception as e:  # noqa: BLE001
        shutil.rmtree(VENV, ignore_errors=True)
        hint = ""
        if platform.system() == "Linux":
            hint = "\nOn Debian/Ubuntu run: sudo apt install python3-venv  (then re-run)"
        die(f"Could not create the virtual environment: {e}{hint}")


def venv_has(*modules: str) -> bool:
    code = ("import importlib.util as u, sys; "
            "sys.exit(0 if all(u.find_spec(m) for m in sys.argv[1:]) else 1)")
    r = capture([VPY, "-c", code, *modules], timeout=60) if VPY.exists() else None
    return bool(r and r[0] == 0)


def pip_install(args: list[str], extra: list[str] | None = None) -> None:
    cmd = [VPY, "-m", "pip", "install", "--disable-pip-version-check", *(extra or []), *args]
    if run_visible(cmd) != 0:
        die(f"pip install {' '.join(args)} failed (see output above).")


def ensure_python_packages(cpu: bool, torch_index: str | None) -> None:
    if venv_has("whisper", "indic_transliteration", "yt_dlp"):
        return
    step("Installing Python packages (openai-whisper, yt-dlp, indic-transliteration). "
         "This downloads PyTorch and can take several minutes.")
    index = torch_index or (CPU_TORCH_INDEX if cpu else None)
    if index and not venv_has("torch"):
        print(f"      installing PyTorch from {index}")
        pip_install(["torch"], ["--index-url", index])
    pip_install(["yt-dlp", "indic-transliteration"])
    # openai-whisper is source-only on PyPI; newer setuptools can break its isolated build.
    cmd = [VPY, "-m", "pip", "install", "--disable-pip-version-check", "openai-whisper"]
    if run_visible(cmd) != 0:
        print("      retrying openai-whisper without build isolation")
        pip_install(["setuptools<81", "wheel"])
        pip_install(["openai-whisper"], ["--no-build-isolation"])
    if not venv_has("whisper", "indic_transliteration", "yt_dlp"):
        die("Python packages did not install correctly.")


SHIM_PY = r'''# Auto-generated by setup_and_run.py.
# cmd/editor calls `whisper ... --device cuda` unconditionally. This wrapper keeps that call
# working on machines without a CUDA GPU by switching the device to cpu.
import os
import sys


def cuda_ok():
    if os.environ.get("VIDEDIT_FORCE_CPU") == "1":
        return False
    try:
        import torch
        return bool(torch.cuda.is_available())
    except Exception:
        return False


def main():
    argv, out, i = sys.argv[1:], [], 0
    while i < len(argv):
        a = argv[i]
        if a == "--device" and i + 1 < len(argv):
            dev = argv[i + 1]
            if dev.startswith("cuda") and not cuda_ok():
                dev = "cpu"
            out += [a, dev]
            i += 2
            continue
        if a.startswith("--device="):
            dev = a.split("=", 1)[1]
            if dev.startswith("cuda") and not cuda_ok():
                dev = "cpu"
            out.append("--device=" + dev)
            i += 1
            continue
        out.append(a)
        i += 1
    from whisper.transcribe import cli
    sys.argv = ["whisper"] + out
    cli()


if __name__ == "__main__":
    main()
'''


def write_shim() -> None:
    SHIM.mkdir(parents=True, exist_ok=True)
    (SHIM / "whisper_shim.py").write_text(SHIM_PY, encoding="utf-8")
    if IS_WIN:
        (SHIM / "whisper.cmd").write_text(
            '@echo off\r\n"%~dp0..\\venv\\Scripts\\python.exe" "%~dp0whisper_shim.py" %*\r\n')
    else:
        p = SHIM / "whisper"
        p.write_text('#!/bin/sh\nd="$(cd "$(dirname "$0")" && pwd)"\n'
                     'exec "$d/../venv/bin/python" "$d/whisper_shim.py" "$@"\n')
        p.chmod(0o755)


def git_ignore_env() -> None:
    exclude = REPO / ".git" / "info" / "exclude"
    try:
        if exclude.parent.is_dir():
            text = exclude.read_text() if exclude.exists() else ""
            if ".videdit-env/" not in text:
                exclude.write_text(text + ("\n" if text and not text.endswith("\n") else "")
                                   + ".videdit-env/\n")
    except OSError:
        pass


# ----------------------------------------------------------------------------- checks
def report(final: bool) -> None:
    """Print the status of every prerequisite (records failures in `problems`)."""
    problems.clear()
    todo.clear()
    v = sys.version_info
    if (v.major, v.minor) >= MIN_PY:
        ok("Python", f"{v.major}.{v.minor}.{v.micro} (runs this script)")
    else:
        bad("Python", f"{v.major}.{v.minor} found, need {MIN_PY[0]}.{MIN_PY[1]}+")

    g = probe_go()
    if g:
        ok("Go", f"{'.'.join(map(str, g[1]))} ({g[0]})")
    else:
        if final:
            bad("Go", f"need {GO_VERSION[2:]}+")
        else:
            warn("Go", f"need {GO_VERSION[2:]}+ -> will download locally")
            todo.append("Go")

    ff = probe_ffmpeg()
    if ff:
        ok("ffmpeg + ffprobe", ff[1])
    else:
        if final:
            bad("ffmpeg + ffprobe", "need libx264 + ass filter")
        else:
            warn("ffmpeg + ffprobe", "need libx264 + ass filter -> will install static build")
            todo.append("ffmpeg")

    pkgs_ok = venv_has("whisper", "indic_transliteration", "yt_dlp")
    if pkgs_ok:
        ok("whisper/yt-dlp/indic", f"installed in {VENV.relative_to(REPO)}")
    else:
        if final:
            bad("whisper/yt-dlp/indic", "not installed")
        else:
            warn("whisper/yt-dlp/indic", "not installed -> will install into venv")
            todo.append("python packages")

    has_nv = shutil.which("nvidia-smi") is not None
    if pkgs_ok and venv_has("torch"):
        r = capture([VPY, "-c", "import torch;print(torch.cuda.is_available())"], timeout=120)
        cuda = bool(r and "True" in r[1])
        ok("Whisper device", "GPU (CUDA)" if cuda else "CPU (automatic fallback via shim)")
        if has_nv and not cuda:
            warn("NVIDIA GPU", "found, but PyTorch has no CUDA. For GPU, reinstall torch with "
                 "--torch-index-url https://download.pytorch.org/whl/cu126 (see pytorch.org)")
    elif has_nv:
        ok("NVIDIA GPU", "detected (nvidia-smi)")

    if shutil.which("node"):
        ok("node (optional)", "found")
    else:
        warn("node (optional)", "not found; only needed if yt-dlp asks for a JS runtime")

    if (REPO / "internal" / "web").is_dir():
        ok("Web UI source", "internal/web present")
    else:
        bad("Web UI source", "internal/web missing -> run: git checkout web-ui")

    if port_in_use(PORT):
        warn(f"Port {PORT}", "already in use; stop the other program before starting")
    else:
        ok(f"Port {PORT}", "free")


# ----------------------------------------------------------------------------- run
def build_env(go_path: str, ffmpeg_dir: Path | None) -> dict:
    env = os.environ.copy()
    prefix = [str(SHIM), str(VENV / BIN)]
    if Path(go_path) == LOCAL_GO:
        prefix.append(str(GO_DIR / "bin"))
    if ffmpeg_dir:
        prefix.append(str(ffmpeg_dir))
    env["PATH"] = os.pathsep.join(prefix + [env.get("PATH", "")])
    env["GOPATH"] = str(ENV / "gopath")
    env["GOCACHE"] = str(ENV / "gocache")
    env["GOTOOLCHAIN"] = "local"          # never auto-download another toolchain
    env["GOFLAGS"] = "-buildvcs=false"    # works even if git is not installed
    env["PYTHONUTF8"] = "1"
    env["PYTHONIOENCODING"] = "utf-8"
    env["VIRTUAL_ENV"] = str(VENV)
    env.pop("PYTHONHOME", None)
    return env


def open_browser_when_ready() -> None:
    for _ in range(240):
        if port_in_use(PORT):
            time.sleep(0.5)
            webbrowser.open(URL)
            return
        time.sleep(0.5)


def main() -> None:
    ap = argparse.ArgumentParser(description="Check, install and run Videdit (web-ui branch).")
    ap.add_argument("--check", action="store_true", help="only report status, install nothing")
    ap.add_argument("--cpu", action="store_true",
                    help="install CPU-only PyTorch (much smaller download; no NVIDIA GPU)")
    ap.add_argument("--torch-index-url", help="custom PyTorch wheel index (e.g. a CUDA build)")
    ap.add_argument("--no-run", action="store_true", help="set up but do not start the server")
    ap.add_argument("--no-browser", action="store_true", help="do not open the browser")
    ap.add_argument("--clean", action="store_true", help="delete .videdit-env and exit")
    ap.add_argument("editor_args", nargs=argparse.REMAINDER,
                    help="after '--': arguments passed to cmd/editor (a video path or URL)")
    args = ap.parse_args()
    editor_args = [a for a in args.editor_args if a != "--"]

    if args.clean:
        shutil.rmtree(ENV, ignore_errors=True)
        print(f"Removed {ENV}")
        return
    if sys.version_info[:2] < MIN_PY:
        die(f"Python {MIN_PY[0]}.{MIN_PY[1]}+ is required (you have {platform.python_version()}).")
    if not (REPO / "go.mod").exists() or not (REPO / "cmd" / "editor").is_dir():
        die(f"{REPO} does not look like the reel-generator repo (go.mod / cmd/editor missing). "
            "Put this file in the repo root.")

    step("Checking prerequisites")
    report(final=False)
    if args.check:
        if todo:
            print("\nStill to install (done automatically by running without --check): "
                  + ", ".join(todo))
        sys.exit(1 if (problems or todo) else 0)

    step("Setting up the isolated environment")
    ensure_venv()
    ensure_python_packages(args.cpu, args.torch_index_url)
    ffmpeg_dir = ensure_ffmpeg()
    if not probe_go():
        install_go()
    write_shim()
    git_ignore_env()

    step("Verifying")
    report(final=True)
    go = probe_go()
    if problems or not go:
        die("Some prerequisites are still missing: " + ", ".join(problems or ["Go"]))
    if args.no_run:
        print("\nSetup complete. Start later with: python setup_and_run.py")
        return
    if port_in_use(PORT):
        die(f"Port {PORT} is already in use. The editor's port is hardcoded; "
            "stop whatever is using it and re-run.")

    env = build_env(go[0], ffmpeg_dir)
    step(f"Starting Videdit -> {URL}   (Ctrl+C to stop)")
    print("      First build takes a moment; the first transcription downloads the Whisper "
          "model (~460 MB).\n", flush=True)
    if not args.no_browser:
        threading.Thread(target=open_browser_when_ready, daemon=True).start()
    try:
        rc = subprocess.call([go[0], "run", "./cmd/editor", *editor_args], cwd=REPO, env=env)
    except KeyboardInterrupt:
        rc = 130
    sys.exit(rc)


if __name__ == "__main__":
    main()
