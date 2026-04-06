#!/usr/bin/env python3
import argparse
import json
from pathlib import Path

import docker


SCRIPT_DIR = Path(__file__).resolve().parent
DEFAULT_CHALLENGE_FILE = SCRIPT_DIR.parent / "terraform" / "challenges.json"
DEFAULT_FLAGS_DIR = SCRIPT_DIR.parent / "flags"


def parse_args():
    parser = argparse.ArgumentParser(
        description="Copy generated flag files into the matching challenge containers."
    )
    parser.add_argument(
        "--flags-dir",
        default=str(DEFAULT_FLAGS_DIR),
        help="Directory containing generated flag files.",
    )
    parser.add_argument(
        "--challenge-file",
        default=str(DEFAULT_CHALLENGE_FILE),
        help="Challenge definition file used by Terraform.",
    )
    return parser.parse_args()


def load_challenges(challenge_file):
    with Path(challenge_file).open("r", encoding="utf-8") as handle:
        challenge_data = json.load(handle)
    return sorted(challenge_data.get("challenges", {}).keys())


def iter_flag_files(flags_dir, challenge_name):
    challenge_dir = flags_dir / challenge_name
    if challenge_dir.exists():
        yield from sorted(challenge_dir.glob("*.txt"))

    yield from sorted(flags_dir.glob(f"{challenge_name}_team*.txt"))


def inject_flag_file(container, flag_file):
    flag_value = flag_file.read_text(encoding="utf-8").strip()
    destination = f"/flags/{flag_file.name}"
    command = f"mkdir -p /flags && cat > {destination!s} <<'EOF'\n{flag_value}\nEOF"
    result = container.exec_run(["sh", "-c", command])
    return result.exit_code, destination


def main():
    args = parse_args()
    flags_dir = Path(args.flags_dir)

    if not flags_dir.exists():
        raise SystemExit(f"Flag directory not found: {flags_dir}")

    client = docker.from_env()
    challenges = load_challenges(args.challenge_file)

    print(f"\nInjecting flags from {flags_dir}...\n")

    for challenge_name in challenges:
        flag_files = list(iter_flag_files(flags_dir, challenge_name))
        if not flag_files:
            print(f"{challenge_name} - no flag files found")
            continue

        try:
            container = client.containers.get(challenge_name)
        except docker.errors.NotFound:
            print(f"{challenge_name} - container not running")
            continue

        print(f"{challenge_name}:")
        for flag_file in flag_files:
            try:
                exit_code, destination = inject_flag_file(container, flag_file)
            except docker.errors.DockerException as error:
                print(f"  FAILED - {flag_file.name}: {error}")
                continue

            if exit_code == 0:
                print(f"  SUCCESS - {flag_file.name} -> {destination}")
            else:
                print(f"  FAILED - {flag_file.name}")
        print()

    print("Done\n")
    return 0


main()