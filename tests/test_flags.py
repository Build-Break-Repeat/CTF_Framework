import json
import subprocess
import sys
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
CREATE_FLAGS_SCRIPT = ROOT_DIR / "scripts" / "createflags.py"
CHALLENGE_FILE = ROOT_DIR / "terraform" / "challenges.json"


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


def load_challenge_names():
    challenge_data = json.loads(CHALLENGE_FILE.read_text(encoding="utf-8"))
    return sorted(challenge_data["challenges"].keys())


def test_createflags_classic_preset_uses_terraform_challenge_ids(tmp_path):
    output_dir, result = run_createflags(tmp_path, "2")

    assert "Creating flags for 2 team(s)" in result.stdout
    assert (output_dir / "dvwa" / "team1.txt").read_text(encoding="utf-8") == "flag:{dvwa_team1}"
    assert (output_dir / "sqli_labs" / "team2.txt").read_text(encoding="utf-8") == "flag:{sqli_labs_team2}"
    assert (output_dir / "metasploitable2" / "team1.txt").exists()

    manifest = json.loads((output_dir / "manifest.json").read_text(encoding="utf-8"))
    assert manifest["preset"] == "classic"
    assert len(manifest["entries"]) == len(load_challenge_names()) * 2


def test_createflags_lab_preset_builds_team_files(tmp_path):
    output_dir, _ = run_createflags(tmp_path, "1", "--preset", "lab")

    flag_file = output_dir / "dvwa" / "team1.txt"
    assert flag_file.read_text(encoding="utf-8") == "BBR{dvwa-team1}"

    manifest = json.loads((output_dir / "manifest.json").read_text(encoding="utf-8"))
    assert manifest["preset"] == "lab"
    assert len(manifest["entries"]) == len(load_challenge_names())