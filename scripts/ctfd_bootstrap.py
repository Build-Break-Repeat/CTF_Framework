import json
import os
import secrets
import string
import subprocess
import sys
import time
import urllib.error
import urllib.request

# =========================
# CONFIG
# =========================
CTFD_CONTAINER = "ctfd"
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
TOKEN_OUTPUT_FILE = os.path.join(BASE_DIR, "terraform", "ctfd_token.txt")
ADMIN_PASSWORD_FILE = os.path.join(BASE_DIR, "admin_password.txt")
CONFIG_FILE = "../config.json"
MAX_RETRIES = 30
SLEEP_INITIAL = 2     # Initial sleep
SLEEP_MAX     = 10    # Max sleep time between checks
SLEEP_FACTOR  = 1.5   # Multiply by this after each failed check
LOG_TAIL_LINES = 20   # Lines to show on failure
DEBUG = True


def debug(msg):
    if DEBUG:
        print(msg)


# =========================
# WAIT FOR CONTAINER
# =========================

def ctfd_http_ready() -> bool:
    try:
        urllib.request.urlopen("http://127.0.0.1:8000", timeout=3)
        return True
    except urllib.error.HTTPError:
        # Any HTTP response (even 4xx/5xx) means CTFd is serving
        return True
    except Exception:
        return False


def dump_logs(name: str):
    print(f"[*] Last {LOG_TAIL_LINES} lines of {name} logs:")
    result = subprocess.run(
        ["docker", "logs", "--tail", str(LOG_TAIL_LINES), name],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    print(result.stdout.decode())



def wait_for_ctfd() -> bool:
    debug("[*] Waiting for CTFd to become ready...")
    sleep = SLEEP_INITIAL
    for i in range(MAX_RETRIES):
        if ctfd_http_ready():
            debug("[+] CTFd is responding")
            return True
        debug(f"Attempt {i + 1}/{MAX_RETRIES} — CTFd not ready, retrying in {sleep:.0f}s")
        time.sleep(sleep)
        sleep = min(sleep * SLEEP_FACTOR, SLEEP_MAX)
    return False


# =========================
# RUN PYTHON INSIDE CONTAINER
# =========================
def run_ctfd_python(script: str) -> str:
    """
    Wraps the given script in a CTFd app context and runs it inside the
    container via `python -` (stdin), avoiding docker cp permission issues.
    Any exception in the script is caught and printed as [EXCEPTION] so
    the caller can detect failures reliably.
    """
    wrapped = f"""
import sys
from CTFd import create_app
from CTFd.models import db

app = create_app()
with app.app_context():
    try:
{chr(10).join("        " + line for line in script.strip().splitlines())}
    except Exception as e:
        print(f"[EXCEPTION] {{type(e).__name__}}: {{e}}", flush=True)
        sys.exit(1)
"""
    result = subprocess.run(
        ["docker", "exec", "-i", CTFD_CONTAINER, "python", "-"],
        input=wrapped.encode(),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    output = result.stdout.decode()
    if DEBUG:
        debug(f"[script stdout]\n{output}")
        stderr = result.stderr.decode()
        if stderr:
            debug(f"[script stderr]\n{stderr}")
    return output


def check_output(output: str, context: str):
    """Exit with a useful message if the script emitted an [EXCEPTION] line."""
    for line in output.splitlines():
        if line.startswith("[EXCEPTION]"):
            print(f"[ERROR] {context} failed: {line}")
            sys.exit(1)


# =========================
# LOAD CONFIG
# =========================
def load_config() -> dict:
    path = os.path.join(os.path.dirname(__file__), CONFIG_FILE)
    with open(path) as f:
        return json.load(f)


def get_admin_credentials(cfg: dict) -> tuple:

    admin_cfg = cfg.get("event", {}).get("admin", {})
    username = admin_cfg.get("username", "admin")
    password = admin_cfg.get("password")

    if not password:
        alphabet = string.ascii_letters + string.digits
        password = "".join(secrets.choice(alphabet) for _ in range(8))
        with open(ADMIN_PASSWORD_FILE, "w") as f:
            f.write(f"username: {username}\npassword: {password}\n")
        print(f"[*] No admin password in config — generated and saved to {ADMIN_PASSWORD_FILE}")

    return username, password


# =========================
# CREATE ADMIN
# =========================
def create_admin(username: str, password: str):
    print("[*] Creating admin user...")
    username_escaped = username.replace("\\", "\\\\").replace('"', '\\"')
    password_escaped = password.replace("\\", "\\\\").replace('"', '\\"')
    script = f"""
from CTFd.models import Users, db

user = Users.query.filter_by(name="{username_escaped}").first()
if not user:
    user = Users(
        name="{username_escaped}",
        email="{username_escaped}@ctf.local",
        password="{password_escaped}",
        type="admin",
        verified=True,
        hidden=True,
    )
    db.session.add(user)
    db.session.commit()
    print("ADMIN_CREATED")
else:
    user.type = "admin"
    user.hidden = True
    db.session.commit()
    print("ADMIN_EXISTS")
"""
    output = run_ctfd_python(script)
    check_output(output, "create_admin")
    if "ADMIN_CREATED" in output:
        print("[+] Admin created")
    elif "ADMIN_EXISTS" in output:
        print("[*] Admin already exists")
    else:
        print("[ERROR] Admin creation failed — no sentinel in output")
        sys.exit(1)


# =========================
# CTFd SETUP 
# =========================
def run_setup(cfg: dict):
    print("[*] Running CTFd setup...")

    event = cfg["event"]
    ctf_name = event["name"]
    # config.json uses an integer team count; CTFd user_mode is "users" or "teams"
    user_mode = "teams" if event.get("teams", 0) > 1 else "users"
    team_size = event.get("teams", None)

    ctf_name_escaped = ctf_name.replace('"', '\\"')

    script = f"""
from CTFd.models import Pages, db
from CTFd.utils import set_config

# General
set_config("ctf_name", "{ctf_name_escaped}")
set_config("ctf_description", "")
set_config("user_mode", "{user_mode}")

# Visibility (match setup-wizard defaults: challenges private until started)
set_config("challenge_visibility", "private")
set_config("registration_visibility", "public")
set_config("score_visibility", "public")
set_config("account_visibility", "public")

# Team size
set_config("team_size", {repr(team_size)})

# Email (disabled - no mail server configured)
set_config("mail_server", None)
set_config("mail_port", None)
set_config("mail_tls", None)
set_config("mail_ssl", None)
set_config("mail_username", None)
set_config("mail_password", None)
set_config("mail_useauth", None)
set_config("verify_emails", None)

# Misc
set_config("social_shares", None)
set_config("start", None)
set_config("end", None)
set_config("freeze", None)

# Index page (required - CTFd errors without it)
existing = Pages.query.filter_by(route="index").first()
if not existing:
    page = Pages(title="{ctf_name_escaped}", route="index", content="", draft=False)
    db.session.add(page)
    db.session.commit()

# Mark setup complete
set_config("setup", True)
print("SETUP_DONE")
"""
    output = run_ctfd_python(script)
    check_output(output, "run_setup")
    if "SETUP_DONE" in output:
        print("[+] CTFd setup complete")
    else:
        print("[ERROR] Setup sentinel not found in output")
        sys.exit(1)



# =========================
# GENERATE TOKEN
# =========================
def generate_token(username: str) -> str:
    print("[*] Generating API token...")
    username_escaped = username.replace("\\", "\\\\").replace('"', '\\"')
    script = f"""
import os
import datetime
from CTFd.models import Users, UserTokens, db
from CTFd.utils.encoding import hexencode

user = Users.query.filter_by(name="{username_escaped}").first()
if not user:
    raise Exception("Admin user not found")

value = "ctfd_" + hexencode(os.urandom(32))
while UserTokens.query.filter_by(value=value).first():
    value = "ctfd_" + hexencode(os.urandom(32))

expiration = datetime.datetime.utcnow() + datetime.timedelta(days=365)
token = UserTokens(
    user_id=user.id,
    value=value,
    expiration=expiration,
    description="Bootstrap admin token",
)
db.session.add(token)
db.session.commit()
print("TOKEN_OUTPUT:" + token.value)
"""
    output = run_ctfd_python(script)
    check_output(output, "generate_token")
    for line in output.splitlines():
        if line.startswith("TOKEN_OUTPUT:"):
            token = line.split("TOKEN_OUTPUT:")[1].strip()
            print(f"[+] Token generated: {token[:12]}...")
            return token
    print("[ERROR] Token sentinel not found in output")
    sys.exit(1)


# =========================
# SAVE TOKEN
# =========================
def save_token(token: str):
    os.makedirs(os.path.dirname(TOKEN_OUTPUT_FILE), exist_ok=True)
    with open(TOKEN_OUTPUT_FILE, "w") as f:
        f.write(token)
    print(f"[+] Token saved to {TOKEN_OUTPUT_FILE}")


# =========================
# MAIN
# =========================
def main():
    if os.path.exists(TOKEN_OUTPUT_FILE):
        print("[*] Token file already exists, skipping bootstrap")
        return

    if not wait_for_ctfd():
        print("[ERROR] CTFd never became ready")
        dump_logs(CTFD_CONTAINER)
        dump_logs(f"{CTFD_CONTAINER}-db")
        sys.exit(1)

    cfg = load_config()
    username, password = get_admin_credentials(cfg)

    create_admin(username, password)
    run_setup(cfg)
    token = generate_token(username)
    save_token(token)
    print("[+] Bootstrap complete")


if __name__ == "__main__":
    main()
