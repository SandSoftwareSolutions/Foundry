import contextlib
import io
import pathlib
import sys
import unittest

# Make the package importable without installing it.
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[1] / "src"))

from {{PROJECT_NAME_SNAKE}}.cli import main  # noqa: E402


class CliTest(unittest.TestCase):
    def run_cli(self, *argv: str) -> str:
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            self.assertEqual(main(list(argv)), 0)
        return out.getvalue()

    def test_default_greeting(self) -> None:
        self.assertIn("Hello, world!", self.run_cli())

    def test_named_greeting(self) -> None:
        self.assertIn("Hello, Ada!", self.run_cli("--name", "Ada"))


if __name__ == "__main__":
    unittest.main()
