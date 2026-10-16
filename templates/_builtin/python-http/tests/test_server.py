import json
import pathlib
import sys
import threading
import unittest
import urllib.error
import urllib.request

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[1] / "src"))

from {{PROJECT_NAME_SNAKE}}.server import make_server  # noqa: E402


class ServerTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.server = make_server("127.0.0.1", 0)  # port 0: let the system pick a free one
        cls.base = f"http://127.0.0.1:{cls.server.server_address[1]}"
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()

    @classmethod
    def tearDownClass(cls) -> None:
        cls.server.shutdown()
        cls.server.server_close()

    def get(self, path: str):
        with urllib.request.urlopen(self.base + path) as response:
            return response.status, json.load(response)

    def test_health(self) -> None:
        self.assertEqual(self.get("/healthz"), (200, {"status": "ok"}))

    def test_root(self) -> None:
        status, body = self.get("/")
        self.assertEqual(status, 200)
        self.assertIn("{{PROJECT_NAME}}", body["message"])

    def test_unknown_path(self) -> None:
        with self.assertRaises(urllib.error.HTTPError) as caught:
            self.get("/nope")
        self.assertEqual(caught.exception.code, 404)


if __name__ == "__main__":
    unittest.main()
