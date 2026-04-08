import json
import subprocess
import sys
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
CREATE_FLAGS_SCRIPT = ROOT_DIR / "scripts" / "createflags.py"
INJECT_FLAGS_SCRIPT = ROOT_DIR / "scripts" / "injectflags.py"
CHALLENGE_FILE = ROOT_DIR / "config.json"


def run_createflags(tmp_path, *args):
    output_dir = tmp_path / "flags"
    command = [
        sys.executable,
        str(CREATE_FLAGS_SCRIPT),
        *args,
        "--output-dir",
        str(output_dir),
        "--challenge-file",
        str(CHALLENGE_FILE),
    ]
    result = subprocess.run(command, cwd=ROOT_DIR, capture_output=True, text=True, check=True)
    return output_dir, result


def run_createflags_raw(tmp_path, *args):
    output_dir = tmp_path / "flags"
    command = [
        sys.executable,
        str(CREATE_FLAGS_SCRIPT),
        *args,
        "--output-dir",
        str(output_dir),
        "--challenge-file",
        str(CHALLENGE_FILE),
    ]
    result = subprocess.run(command, cwd=ROOT_DIR, capture_output=True, text=True)
    return output_dir, result


def run_injectflags_raw(flags_dir, *args):
    command = [
        sys.executable,
        str(INJECT_FLAGS_SCRIPT),
        "--flags-dir",
        str(flags_dir),
        "--challenge-file",
        str(CHALLENGE_FILE),
        *args,
    ]
    return subprocess.run(command, cwd=ROOT_DIR, capture_output=True, text=True)


def load_challenge_names():
    challenge_data = json.loads(CHALLENGE_FILE.read_text(encoding="utf-8"))
    challenges = challenge_data["challenges"]

    if isinstance(challenges, dict):
        return sorted(challenges.keys())

    return sorted(challenge["id"] for challenge in challenges if "id" in challenge)


def read_flag(output_dir, challenge_name, team_number):
    return (output_dir / challenge_name / f"team{team_number}.txt").read_text(encoding="utf-8")


def test_createflags_classic_preset_uses_terraform_challenge_ids(tmp_path):
    output_dir, result = run_createflags(tmp_path, "2")
    challenge_names = load_challenge_names()

    assert "Creating flags for 2 team(s)" in result.stdout
    assert challenge_names
    assert read_flag(output_dir, challenge_names[0], 1) == f"flag:{{{challenge_names[0]}_team1}}"
    assert read_flag(output_dir, challenge_names[-1], 2) == f"flag:{{{challenge_names[-1]}_team2}}"

    manifest = json.loads((output_dir / "manifest.json").read_text(encoding="utf-8"))
    assert manifest["preset"] == "classic"
    assert len(manifest["entries"]) == len(challenge_names) * 2


def test_createflags_lab_preset_builds_team_files(tmp_path):
    output_dir, _ = run_createflags(tmp_path, "1", "--preset", "lab")
    challenge_names = load_challenge_names()
    first_challenge = challenge_names[0]

    assert read_flag(output_dir, first_challenge, 1) == f"BBR{{{first_challenge}-team1}}"

    manifest = json.loads((output_dir / "manifest.json").read_text(encoding="utf-8"))
    assert manifest["preset"] == "lab"
    assert len(manifest["entries"]) == len(challenge_names)


def test_createflags_rejects_invalid_preset(tmp_path):
    _, result = run_createflags_raw(tmp_path, "1", "--preset", "does-not-exist")

    assert result.returncode != 0
    assert "Unknown preset" in result.stderr


def test_createflags_rejects_team_count_below_one(tmp_path):
    _, result = run_createflags_raw(tmp_path, "0")

    assert result.returncode != 0
    assert "team_count must be at least 1" in result.stderr


def test_createflags_lists_available_presets(tmp_path):
    _, result = run_createflags_raw(tmp_path, "--list-presets")

    assert result.returncode == 0
    assert "classic" in result.stdout
    assert "lab" in result.stdout


def test_injectflags_fails_when_flags_dir_is_missing(tmp_path):
    missing_dir = tmp_path / "missing-flags"
    result = run_injectflags_raw(missing_dir)

    assert result.returncode != 0
    assert "Flag directory not found" in result.stderr
