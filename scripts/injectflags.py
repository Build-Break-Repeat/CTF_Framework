#!/usr/bin/env python3
import argparse
import json
from pathlib import Path

import docker


SCRIPT_DIR = Path(__file__).resolve().parent
DEFAULT_CHALLENGE_FILE = SCRIPT_DIR.parent / "terraform" / "config.json"
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

    challenges = challenge_data.get("challenges", {})

    if isinstance(challenges, dict):
        return dict(sorted(challenges.items()))

    if isinstance(challenges, list):
        return {
            challenge["id"]: challenge
            for challenge in sorted(challenges, key=lambda challenge: challenge.get("id", ""))
            if "id" in challenge
        }

    raise SystemExit("Invalid challenge format in challenge file")


def iter_flag_files(flags_dir, challenge_name):
    challenge_dir = flags_dir / challenge_name
    if challenge_dir.exists():
        yield from sorted(challenge_dir.glob("*.txt"))

    yield from sorted(flags_dir.glob(f"{challenge_name}_team*.txt"))


def get_team_number(flag_file):
    name = flag_file.stem
    if "team" not in name:
        return 1

    team_text = name.split("team", 1)[1]
    digits = ""
    for char in team_text:
        if char.isdigit():
            digits += char
        else:
            break

    if digits:
        return int(digits)

    return 1


def get_destination(challenge_name, challenge_config, flag_file):
    flag_config = challenge_config.get("flag", {})
    path_template = flag_config.get("path")

    if not path_template:
        return f"/flags/{flag_file.name}"

    destination = path_template.format(
        challenge=challenge_name,
        team=get_team_number(flag_file),
        file=flag_file.name,
    )

    if destination.endswith("/"):
        return f"{destination}{flag_file.name}"

    return destination


def inject_flag_file(container, challenge_name, challenge_config, flag_file):
    flag_value = flag_file.read_text(encoding="utf-8").strip()
    destination = get_destination(challenge_name, challenge_config, flag_file)
    flag_config = challenge_config.get("flag", {})
    destination_dir = str(Path(destination).parent).replace("\\", "/")
    command = f"mkdir -p '{destination_dir}' && cat > '{destination}' <<'EOF'\n{flag_value}\nEOF"

    if flag_config.get("permissions"):
        command += f" && chmod {flag_config['permissions']} '{destination}'"

    if flag_config.get("owner"):
        command += f" && chown {flag_config['owner']} '{destination}'"

    result = container.exec_run(["sh", "-c", command])
    return result.exit_code, destination


def main():
    args = parse_args()
    flags_dir = Path(args.flags_dir)

    if not flags_dir.exists():
        raise SystemExit(f"Flag directory not found: {flags_dir}")

    challenges = load_challenges(args.challenge_file)
    client = docker.from_env()

    print(f"\nInjecting flags from {flags_dir}...\n")

    for challenge_name, challenge_config in challenges.items():
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
                exit_code, destination = inject_flag_file(container, challenge_name, challenge_config, flag_file)
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