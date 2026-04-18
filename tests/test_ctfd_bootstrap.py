"""
Unit tests for scripts/ctfd_bootstrap.py

All Docker and network calls are mocked so these tests run without
a live Docker daemon or CTFd instance.
"""

import importlib.util
import json
import os
import sys
import types
import unittest
from pathlib import Path
from unittest.mock import MagicMock, patch, mock_open, call

# ---------------------------------------------------------------------------
# Load the module under test without executing main()
# ---------------------------------------------------------------------------

SCRIPT_PATH = Path(__file__).parent.parent / "scripts" / "ctfd_bootstrap.py"


def load_bootstrap_module():
    """Import ctfd_bootstrap as a module without executing main()."""
    spec = importlib.util.spec_from_file_location("ctfd_bootstrap", SCRIPT_PATH)
    mod = importlib.util.module_from_spec(spec)
    sys.modules["ctfd_bootstrap"] = mod
    # The `if __name__ == "__main__"` guard prevents main() from running
    # when loaded as a module (mod.__name__ is "ctfd_bootstrap", not "__main__")
    spec.loader.exec_module(mod)
    return mod


bs = load_bootstrap_module()


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

MINIMAL_CONFIG = {
    "event": {
        "name": "Test CTF",
        "max_team_size": 2,
        "admin": {"username": "admin", "password": "s3cr3t"},
    },
    "challenges": [],
}


# ---------------------------------------------------------------------------
# ctfd_http_ready
# ---------------------------------------------------------------------------

class TestCtfdHttpReady(unittest.TestCase):

    @patch("urllib.request.urlopen")
    def test_returns_true_on_200(self, mock_urlopen):
        mock_urlopen.return_value = MagicMock()
        self.assertTrue(bs.ctfd_http_ready())

    @patch("urllib.request.urlopen")
    def test_returns_true_on_http_error(self, mock_urlopen):
        import urllib.error
        mock_urlopen.side_effect = urllib.error.HTTPError(
            url=None, code=404, msg="Not Found", hdrs=None, fp=None
        )
        self.assertTrue(bs.ctfd_http_ready())

    @patch("urllib.request.urlopen")
    def test_returns_false_on_connection_error(self, mock_urlopen):
        mock_urlopen.side_effect = Exception("Connection refused")
        self.assertFalse(bs.ctfd_http_ready())


# ---------------------------------------------------------------------------
# wait_for_ctfd
# ---------------------------------------------------------------------------

class TestWaitForCtfd(unittest.TestCase):

    @patch.object(bs, "ctfd_http_ready", return_value=True)
    @patch("time.sleep")
    def test_returns_true_when_ready_immediately(self, mock_sleep, mock_ready):
        self.assertTrue(bs.wait_for_ctfd())
        mock_sleep.assert_not_called()

    @patch.object(bs, "ctfd_http_ready", side_effect=[False, False, True])
    @patch("time.sleep")
    def test_returns_true_after_retries(self, mock_sleep, mock_ready):
        self.assertTrue(bs.wait_for_ctfd())
        self.assertEqual(mock_sleep.call_count, 2)

    @patch.object(bs, "ctfd_http_ready", return_value=False)
    @patch("time.sleep")
    def test_returns_false_after_max_retries(self, mock_sleep, mock_ready):
        result = bs.wait_for_ctfd()
        self.assertFalse(result)
        self.assertEqual(mock_ready.call_count, bs.MAX_RETRIES)


# ---------------------------------------------------------------------------
# check_output
# ---------------------------------------------------------------------------

class TestCheckOutput(unittest.TestCase):

    def test_no_exception_in_output_does_not_exit(self):
        bs.check_output("SETUP_DONE\nADMIN_CREATED\n", "test")

    def test_exception_in_output_exits(self):
        with self.assertRaises(SystemExit):
            bs.check_output("[EXCEPTION] ValueError: something went wrong", "test")

    def test_exception_mid_line_does_not_exit(self):
        # check_output uses startswith so [EXCEPTION] mid-line is not matched
        bs.check_output("prefix [EXCEPTION] RuntimeError: oops", "test")

    def test_multiple_lines_clean(self):
        bs.check_output("line1\nline2\nSETUP_DONE", "test")


# ---------------------------------------------------------------------------
# load_config
# ---------------------------------------------------------------------------

class TestLoadConfig(unittest.TestCase):

    def test_returns_parsed_dict(self, *_):
        with patch("builtins.open", mock_open(read_data=json.dumps(MINIMAL_CONFIG))):
            cfg = bs.load_config()
        self.assertEqual(cfg["event"]["name"], "Test CTF")
        self.assertEqual(cfg["challenges"], [])

    def test_raises_on_missing_file(self):
        with patch.object(bs, "CONFIG_FILE", "/nonexistent/path/config.json"):
            with self.assertRaises(FileNotFoundError):
                bs.load_config()


# ---------------------------------------------------------------------------
# get_admin_credentials
# ---------------------------------------------------------------------------

class TestGetAdminCredentials(unittest.TestCase):

    def test_returns_configured_credentials(self):
        cfg = {
            "event": {"admin": {"username": "ctfadmin", "password": "mypass123"}}
        }
        username, password = bs.get_admin_credentials(cfg)
        self.assertEqual(username, "ctfadmin")
        self.assertEqual(password, "mypass123")

    def test_generates_password_when_missing(self):
        cfg = {"event": {"admin": {"username": "admin"}}}
        m = mock_open()
        with patch("builtins.open", m):
            username, password = bs.get_admin_credentials(cfg)
        self.assertEqual(username, "admin")
        self.assertIsNotNone(password)
        self.assertGreater(len(password), 0)

    def test_generated_password_is_alphanumeric(self):
        cfg = {"event": {"admin": {"username": "admin"}}}
        m = mock_open()
        with patch("builtins.open", m):
            _, password = bs.get_admin_credentials(cfg)
        self.assertTrue(password.isalnum(), f"password {password!r} is not alphanumeric")

    def test_defaults_username_to_admin(self):
        cfg = {"event": {"admin": {"password": "pass"}}}
        username, _ = bs.get_admin_credentials(cfg)
        self.assertEqual(username, "admin")

    def test_empty_admin_section(self):
        cfg = {"event": {}}
        m = mock_open()
        with patch("builtins.open", m):
            username, password = bs.get_admin_credentials(cfg)
        self.assertEqual(username, "admin")
        self.assertIsNotNone(password)


# ---------------------------------------------------------------------------
# run_ctfd_python
# ---------------------------------------------------------------------------

class TestRunCtfdPython(unittest.TestCase):

    @patch("subprocess.run")
    def test_returns_stdout_output(self, mock_run):
        mock_run.return_value = MagicMock(
            stdout=b"ADMIN_CREATED\n",
            stderr=b"",
        )
        output = bs.run_ctfd_python("print('ADMIN_CREATED')")
        self.assertIn("ADMIN_CREATED", output)

    @patch("subprocess.run")
    def test_wraps_script_in_app_context(self, mock_run):
        mock_run.return_value = MagicMock(stdout=b"", stderr=b"")
        bs.run_ctfd_python("x = 1")
        call_args = mock_run.call_args
        stdin_input = call_args.kwargs.get("input") or call_args[1].get("input", b"")
        decoded = stdin_input.decode()
        self.assertIn("create_app", decoded)
        self.assertIn("app_context", decoded)

    @patch("subprocess.run")
    def test_passes_script_to_docker_exec(self, mock_run):
        mock_run.return_value = MagicMock(stdout=b"", stderr=b"")
        bs.run_ctfd_python("pass")
        cmd = mock_run.call_args[0][0]
        self.assertIn("docker", cmd)
        self.assertIn("exec", cmd)
        self.assertIn(bs.CTFD_CONTAINER, cmd)


# ---------------------------------------------------------------------------
# generate_token (output parsing)
# ---------------------------------------------------------------------------

class TestGenerateToken(unittest.TestCase):

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_extracts_token_from_output(self, mock_check, mock_run):
        mock_run.return_value = "TOKEN_OUTPUT:ctfd_abc123def456\n"
        token = bs.generate_token("admin")
        self.assertEqual(token, "ctfd_abc123def456")

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_exits_when_sentinel_missing(self, mock_check, mock_run):
        mock_run.return_value = "some output without sentinel\n"
        with self.assertRaises(SystemExit):
            bs.generate_token("admin")

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_token_with_whitespace_is_stripped(self, mock_check, mock_run):
        mock_run.return_value = "TOKEN_OUTPUT:ctfd_mytoken  \n"
        token = bs.generate_token("admin")
        self.assertEqual(token, "ctfd_mytoken")


# ---------------------------------------------------------------------------
# save_token
# ---------------------------------------------------------------------------

class TestSaveToken(unittest.TestCase):

    def test_writes_token_to_file(self):
        m = mock_open()
        with patch("builtins.open", m), patch("os.makedirs"):
            bs.save_token("ctfd_testtoken")
        handle = m()
        handle.write.assert_called_once_with("ctfd_testtoken")

    def test_creates_parent_directory(self):
        m = mock_open()
        with patch("builtins.open", m), patch("os.makedirs") as mock_makedirs:
            bs.save_token("ctfd_testtoken")
        mock_makedirs.assert_called_once()
        call_kwargs = mock_makedirs.call_args
        self.assertTrue(call_kwargs.kwargs.get("exist_ok", False))


# ---------------------------------------------------------------------------
# create_admin (sentinel detection)
# ---------------------------------------------------------------------------

class TestCreateAdmin(unittest.TestCase):

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_reports_admin_created(self, mock_check, mock_run):
        mock_run.return_value = "ADMIN_CREATED\n"
        bs.create_admin("admin", "pass")

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_reports_admin_exists(self, mock_check, mock_run):
        mock_run.return_value = "ADMIN_EXISTS\n"
        bs.create_admin("admin", "pass")

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_exits_when_no_sentinel(self, mock_check, mock_run):
        mock_run.return_value = "unexpected output\n"
        with self.assertRaises(SystemExit):
            bs.create_admin("admin", "pass")

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_escapes_username_quotes(self, mock_check, mock_run):
        mock_run.return_value = "ADMIN_CREATED\n"
        bs.create_admin('admin"evil', "pass")
        script_arg = mock_run.call_args[0][0]
        self.assertNotIn('"admin"evil"', script_arg)

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_escapes_password_quotes(self, mock_check, mock_run):
        mock_run.return_value = "ADMIN_CREATED\n"
        bs.create_admin("admin", 'pass"word')
        script_arg = mock_run.call_args[0][0]
        self.assertNotIn('"pass"word"', script_arg)


# ---------------------------------------------------------------------------
# run_setup (sentinel detection)
# ---------------------------------------------------------------------------

class TestRunSetup(unittest.TestCase):

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_succeeds_with_sentinel(self, mock_check, mock_run):
        mock_run.return_value = "SETUP_DONE\n"
        bs.run_setup(MINIMAL_CONFIG)

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_exits_when_sentinel_missing(self, mock_check, mock_run):
        mock_run.return_value = "something else\n"
        with self.assertRaises(SystemExit):
            bs.run_setup(MINIMAL_CONFIG)

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_team_mode_for_large_teams(self, mock_check, mock_run):
        mock_run.return_value = "SETUP_DONE\n"
        cfg = {
            "event": {"name": "Team CTF", "max_team_size": 4}
        }
        bs.run_setup(cfg)
        script_arg = mock_run.call_args[0][0]
        self.assertIn('"teams"', script_arg)

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_user_mode_for_solo(self, mock_check, mock_run):
        mock_run.return_value = "SETUP_DONE\n"
        cfg = {
            "event": {"name": "Solo CTF", "max_team_size": 1}
        }
        bs.run_setup(cfg)
        script_arg = mock_run.call_args[0][0]
        self.assertIn('"users"', script_arg)

    @patch.object(bs, "run_ctfd_python")
    @patch.object(bs, "check_output")
    def test_escapes_ctf_name_quotes(self, mock_check, mock_run):
        mock_run.return_value = "SETUP_DONE\n"
        cfg = {"event": {"name": 'My "Quoted" CTF', "max_team_size": 1}}
        bs.run_setup(cfg)
        script_arg = mock_run.call_args[0][0]
        self.assertNotIn('"My "Quoted" CTF"', script_arg)


# ---------------------------------------------------------------------------
# main — high level integration of individual steps
# ---------------------------------------------------------------------------

class TestMain(unittest.TestCase):

    @patch.object(bs, "save_token")
    @patch.object(bs, "generate_token", return_value="ctfd_tok")
    @patch.object(bs, "run_setup")
    @patch.object(bs, "create_admin")
    @patch.object(bs, "get_admin_credentials", return_value=("admin", "pass"))
    @patch.object(bs, "load_config", return_value=MINIMAL_CONFIG)
    @patch.object(bs, "wait_for_ctfd", return_value=True)
    @patch("os.path.exists", return_value=False)
    def test_full_bootstrap_flow(
        self, mock_exists, mock_wait, mock_load, mock_creds,
        mock_create, mock_setup, mock_gen, mock_save
    ):
        bs.main()
        mock_wait.assert_called_once()
        mock_load.assert_called_once()
        mock_creds.assert_called_once_with(MINIMAL_CONFIG)
        mock_create.assert_called_once_with("admin", "pass")
        mock_setup.assert_called_once_with(MINIMAL_CONFIG)
        mock_gen.assert_called_once_with("admin")
        mock_save.assert_called_once_with("ctfd_tok")

    @patch("os.path.exists", return_value=True)
    def test_skips_bootstrap_if_token_exists(self, mock_exists):
        with patch.object(bs, "wait_for_ctfd") as mock_wait:
            bs.main()
            mock_wait.assert_not_called()

    @patch.object(bs, "dump_logs")
    @patch.object(bs, "wait_for_ctfd", return_value=False)
    @patch("os.path.exists", return_value=False)
    def test_exits_when_ctfd_never_ready(self, mock_exists, mock_wait, mock_dump):
        with self.assertRaises(SystemExit):
            bs.main()
        mock_dump.assert_called()


if __name__ == "__main__":
    unittest.main()
