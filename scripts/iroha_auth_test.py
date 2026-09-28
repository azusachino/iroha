import unittest.mock
import json
import threading
import unittest
import urllib.request
from http.server import BaseHTTPRequestHandler, HTTPServer

import iroha_auth


class FakeIroha(BaseHTTPRequestHandler):
    owner_exists = True
    seen: list[dict] = []

    def log_message(self, *args):
        pass

    def _reply(self, status, body, cookie=None):
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        if cookie:
            self.send_header("Set-Cookie", f"iroha_session={cookie}; Path=/; HttpOnly; Secure; SameSite=Lax")
        self.end_headers()
        self.wfile.write(json.dumps(body).encode())

    def do_GET(self):
        FakeIroha.seen.append({"method": "GET", "path": self.path, "headers": {k.lower(): v for k, v in self.headers.items()}})
        if self.path == "/api/v1/auth/session":
            self._reply(200, {"setup_required": not FakeIroha.owner_exists, "authenticated": False})
        else:
            self._reply(200, {})

    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(length) or b"{}")
        FakeIroha.seen.append({"method": "POST", "path": self.path, "headers": {k.lower(): v for k, v in self.headers.items()}})
        if self.path in ("/api/v1/auth/login", "/api/v1/auth/setup"):
            if body.get("password") != "right":
                self._reply(401, {"code": "invalid_credentials", "message": "no"})
                return
            FakeIroha.owner_exists = True
            self._reply(200, {"authenticated": True, "csrf_token": "csrf-9"}, cookie="tok-1")
        else:
            self._reply(200, {})


class IrohaAuthTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = HTTPServer(("127.0.0.1", 0), FakeIroha)
        cls.base = f"http://127.0.0.1:{cls.server.server_port}"
        threading.Thread(target=cls.server.serve_forever, daemon=True).start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()

    def setUp(self):
        FakeIroha.seen = []
        FakeIroha.owner_exists = True
        urllib.request.install_opener(urllib.request.build_opener())

    def test_login_then_requests_carry_cookie_and_csrf_on_writes_only(self):
        session = iroha_auth.login(self.base, "owner", "right")
        self.assertEqual(session, iroha_auth.OwnerSession(token="tok-1", csrf_token="csrf-9"))
        iroha_auth.install_urllib(session)
        urllib.request.urlopen(self.base + "/api/v1/metrics").read()
        request = urllib.request.Request(self.base + "/api/v1/tasks", data=b"{}", method="POST")
        urllib.request.urlopen(request).read()
        get, post = FakeIroha.seen[-2], FakeIroha.seen[-1]
        self.assertEqual(get["headers"].get("cookie"), "iroha_session=tok-1")
        self.assertNotIn("x-csrf-token", get["headers"])
        self.assertEqual(post["headers"].get("x-csrf-token"), "csrf-9")

    def test_never_creates_an_owner_unless_allowed(self):
        FakeIroha.owner_exists = False
        with self.assertRaises(RuntimeError):
            iroha_auth.login(self.base, "owner", "right")
        self.assertFalse(any(call["path"] == "/api/v1/auth/setup" for call in FakeIroha.seen))
        iroha_auth.login(self.base, "owner", "right", allow_setup=True)
        self.assertTrue(any(call["path"] == "/api/v1/auth/setup" for call in FakeIroha.seen))

    def test_wrong_password_raises(self):
        with self.assertRaises(OSError):
            iroha_auth.login(self.base, "owner", "wrong")

    def test_credentials_come_from_the_environment(self):
        with unittest.mock.patch.dict("os.environ", {}, clear=True):
            with self.assertRaises(RuntimeError):
                iroha_auth.credentials_from_env()


if __name__ == "__main__":
    unittest.main()
