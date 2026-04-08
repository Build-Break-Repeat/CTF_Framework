#!/usr/bin/env python3
import argparse
import json
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent
DEFAULT_CHALLENGE_FILE = SCRIPT_DIR.parent / "challenges.json"
DEFAULT_PRESET_FILE = SCRIPT_DIR / "flag_presets.json"
DEFAULT_OUTPUT_DIR = SCRIPT_DIR.parent / "flags"


def parse_args():
    parser = argparse.ArgumentParser(
        description="Generate simple CTF flag files for teams."
    )
    parser.add_argument("team_count", nargs="?", type=int, default=1)
    parser.add_argument(
        "--preset",
        default="classic",
        help="Preset name from scripts/flag_presets.json.",
    )
    parser.add_argument(
        "--output-dir",
        default=str(DEFAULT_OUTPUT_DIR),
        help="Directory where flag files will be written.",
    )
    parser.add_argument(
        "--challenge-file",
        default=str(DEFAULT_CHALLENGE_FILE),
        help="Challenge definition file used by Terraform.",
    )
    parser.add_argument(
        "--preset-file",
        default=str(DEFAULT_PRESET_FILE),
        help="JSON file containing available flag presets.",
    )
    parser.add_argument(
        "--list-presets",
        action="store_true",
        help="Print available presets and exit.",
    )
    return parser.parse_args()


def load_json_file(file_path):
    with Path(file_path).open("r", encoding="utf-8") as handle:
        return json.load(handle)


def load_challenges(challenge_file):
    challenge_data = load_json_file(challenge_file)
    challenges = challenge_data.get("challenges", {})

    if isinstance(challenges, dict):
        return sorted(challenges.keys())

    if isinstance(challenges, list):
        return sorted(challenge["id"] for challenge in challenges if "id" in challenge)

    raise SystemExit("Invalid challenge format in challenge file")


def load_presets(preset_file):
    preset_data = load_json_file(preset_file)
    return preset_data.get("presets", {})


def build_flag(template, challenge, team):
    return template.format(
        challenge=challenge,
        team=team,
    )


def write_flag(flag_path, flag_value):
    flag_path.parent.mkdir(parents=True, exist_ok=True)
    flag_path.write_text(flag_value, encoding="utf-8")


def create_team_flags(challenges, output_dir, template, team_count):
    manifest = []
    for team in range(1, team_count + 1):
        print(f"Team {team}:")
        for challenge in challenges:
            flag_value = build_flag(template, challenge, team)
            flag_path = output_dir / challenge / f"team{team}.txt"
            write_flag(flag_path, flag_value)
            print(f"  {challenge} -> {flag_value}")
            manifest.append(
                {
                    "challenge": challenge,
                    "team": team,
                    "file": str(flag_path.relative_to(output_dir)),
                    "flag": flag_value,
                }
            )
        print()
    return manifest


def write_manifest(output_dir, preset_name, manifest):
    manifest_path = output_dir / "manifest.json"
    manifest_path.write_text(
        json.dumps({"preset": preset_name, "entries": manifest}, indent=2),
        encoding="utf-8",
    )


def main():
    args = parse_args()
    presets = load_presets(args.preset_file)

    if args.list_presets:
        print("Available presets:")
        for preset_name, preset_config in sorted(presets.items()):
            print(f"  {preset_name}: {preset_config.get('description', 'No description')}" )
        return 0

    if args.team_count < 1:
        raise SystemExit("team_count must be at least 1")

    if args.preset not in presets:
        raise SystemExit(f"Unknown preset: {args.preset}")

    preset = presets[args.preset]
    challenges = load_challenges(args.challenge_file)
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    print(f"\nCreating flags for {args.team_count} team(s) using preset '{args.preset}'...\n")

    manifest = create_team_flags(
        challenges,
        output_dir,
        preset["template"],
        args.team_count,
    )

    write_manifest(output_dir, args.preset, manifest)
    print(f"Created {len(manifest)} flag file(s) in {output_dir}\n")
    return 0


main()
