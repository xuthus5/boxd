#!/usr/bin/env python3
"""Validate an already-built boxd image using isolated, task-owned Podman containers."""

import argparse
import hashlib
import json
import os
import secrets
import socket
import struct
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import uuid
from pathlib import Path


def command(*args):
    result = subprocess.run(args, check=True, capture_output=True, text=True)
    return result.stdout.strip()


class Client:
    def __init__(self, address, password):
        self.url = "http://" + address
        self.opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
        self.token = None
        for _ in range(100):
            try:
                status, login = self.call("/api/auth/login", "POST", {
                    "username": "admin", "password": password,
                })
                if status == 200:
                    self.token = login["data"]["token"]
                    return
            except OSError:
                pass
            time.sleep(0.2)
        raise RuntimeError("container did not become ready for login")

    def call(self, path, method="GET", body=None):
        headers = {"Content-Type": "application/json"}
        if self.token:
            headers["Authorization"] = "Bearer " + self.token
        data = json.dumps(body).encode() if body is not None else None
        if data is None and method == "POST":
            data = b""
        request = urllib.request.Request(self.url + path, data=data, headers=headers, method=method)
        try:
            with self.opener.open(request, timeout=15) as response:
                return response.status, json.load(response)
        except urllib.error.HTTPError as error:
            return error.code, json.loads(error.read())

    def ok(self, path, method="GET", body=None):
        status, response = self.call(path, method, body)
        if status != 200 or response.get("status") != "ok":
            raise RuntimeError(path + " failed: " + str(response.get("error")))
        return response.get("data")


class Lab:
    def __init__(self, image, directory):
        self.image = image
        self.directory = Path(directory)
        self.prefix = "boxd-smoke-" + secrets.token_hex(4)
        self.containers = []
        self.password = secrets.token_urlsafe(24)
        self.env = self.directory / "container.env"
        self.env.write_text("BOXD_PASSWORD=" + self.password + "\nBOXD_LISTEN=[::]:9091\n")
        self.env.chmod(0o600)

    def start(self, case, initial=None, extra=()):
        root = self.directory / case
        (root / "data").mkdir(mode=0o700, parents=True)
        (root / "config").mkdir(mode=0o700)
        root.chmod(0o700)
        if initial is not None:
            path = root / "config/config.json"
            path.write_text(initial)
            path.chmod(0o600)
        name = self.prefix + "-" + case
        args = ["podman", "run", "-d", "--http-proxy=false", "--name", name,
                "--label", "io.boxd.test=" + self.prefix, "--env-file", str(self.env),
                "-p", "127.0.0.1::9091", "-p", "127.0.0.1::1080",
                "-v", str(root / "data") + ":/var/lib/boxd:Z",
                "-v", str(root / "config") + ":/etc/sing-box:Z"]
        self.containers.append(name)
        command(*args, *extra, self.image)
        address = command("podman", "port", name, "9091/tcp")
        proxy = command("podman", "port", name, "1080/tcp")
        return name, Client(address, self.password), proxy, root

    def close(self):
        failures = []
        for name in reversed(self.containers):
            result = subprocess.run(["podman", "rm", "--ignore", "-f", name], capture_output=True, text=True)
            if result.returncode:
                failures.append(name)
        if failures:
            raise RuntimeError("could not clean up task containers: " + ", ".join(failures))


def handshake(address):
    host, port = address.rsplit(":", 1)
    with socket.create_connection((host, int(port)), timeout=3) as connection:
        connection.sendall(b"\x05\x01\x00")
        if connection.recv(2) != b"\x05\x00":
            raise RuntimeError("published SOCKS listener is unavailable")


def verify_first_start(client, proxy, root):
    status = client.ok("/api/config/setup")
    assert status["kernel_running"], "new default kernel did not start"
    assert all(item["state"] == "ready" for item in status["modules"]), status["modules"]
    assert status["capabilities"]["container"]
    assert status["listeners"][0]["listen"] == "0.0.0.0"
    assert len(status["listeners"]) == 1, "default proxy mode unexpectedly created TUN"
    handshake(proxy)
    assert (root / "config/config.json").stat().st_mode & 0o777 == 0o600


def make_node(lab):
    identity = str(uuid.uuid4())
    config = {
        "log": {"level": "warn"},
        "inbounds": [{"type": "vless", "tag": "fixture", "listen": "0.0.0.0",
                      "listen_port": 21000, "users": [{"uuid": identity}]}],
        "outbounds": [{"type": "direct", "tag": "direct"}],
        "dns": {"servers": [{"type": "https", "tag": "bootstrap", "server": "223.5.5.5"}],
                "final": "bootstrap"},
        "route": {"final": "direct", "default_domain_resolver": "bootstrap"},
    }
    name, client, _, _ = lab.start("node", json.dumps(config))
    client.ok("/api/service/start", "POST")
    address = command("podman", "inspect", name, "--format",
                      "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}")
    return {"tag": "podman-validation-node", "type": "vless", "server": address,
            "port": 21000, "config": {"uuid": identity}}


def verify_import_and_proxy(client, proxy, node):
    client.ok("/api/service/stop", "POST")
    client.ok("/api/import/save", "POST", node)
    assert not client.ok("/api/service/status")["running"], "import unexpectedly started the kernel"
    client.ok("/api/service/start", "POST")
    output = command("curl", "-q", "--proxy", "socks5h://" + proxy, "--silent",
                     "--show-error", "--fail", "--max-time", "20", "--output", "/dev/null",
                     "--write-out", "%{http_code} %{ssl_verify_result}",
                     "https://www.gstatic.com/generate_204")
    assert output == "204 0", "proxy HTTPS or TLS validation failed"


def verify_stale_preview(client):
    request = {"modules": ["inbounds"], "inbound_mode": "proxy", "ipv6": "auto"}
    preview = client.ok("/api/config/setup/preview", "POST", request)
    config = client.ok("/api/config/")
    config["log"]["level"] = "debug"
    client.ok("/api/config/", "PUT", config)
    request["source_hash"] = preview["source_hash"]
    status, response = client.call("/api/config/setup/apply", "POST", request)
    assert status == 409 and response["error"]["code"] == "config_changed"


def verify_failed_apply(client, proxy):
    original = client.ok("/api/config/")
    candidate = json.loads(json.dumps(original))
    candidate["inbounds"][0]["listen_port"] = 9091
    status, response = client.call("/api/config/", "PUT", candidate)
    assert status == 200 and response["status"] == "rolled_back"
    assert response["error"]["code"] == "config_restart_failed"
    assert client.ok("/api/config/") == original, "failed apply did not restore the original config"
    assert client.ok("/api/service/status")["running"]
    handshake(proxy)


def receive_exact(connection, size):
    body = bytearray()
    while len(body) < size:
        chunk = connection.recv(size - len(body))
        if not chunk:
            raise RuntimeError("proxy DNS connection closed before its response")
        body.extend(chunk)
    return bytes(body)


def verify_dns(proxy):
    host, port = proxy.rsplit(":", 1)
    with socket.create_connection((host, int(port)), timeout=15) as connection:
        connection.sendall(b"\x05\x01\x00")
        assert receive_exact(connection, 2) == b"\x05\x00"
        connection.sendall(b"\x05\x01\x00\x01" + socket.inet_aton("8.8.8.8") + struct.pack("!H", 53))
        header = receive_exact(connection, 4)
        assert header[1] == 0, "proxy rejected the DNS request"
        size = {1: 4, 4: 16}.get(header[3])
        if size is None:
            size = receive_exact(connection, 1)[0]
        receive_exact(connection, size + 2)
        # 该域名走默认国内 DoH 分流，测试不依赖实验出口直连 Google DNS。
        question = b"\x03www\x05baidu\x03com\x00" + struct.pack("!HH", 1, 1)
        query = struct.pack("!HHHHHH", 0xB0D1, 0x0100, 1, 0, 0, 0) + question
        connection.sendall(struct.pack("!H", len(query)) + query)
        size = struct.unpack("!H", receive_exact(connection, 2))[0]
        reply = receive_exact(connection, size)
        ident, flags, _, answers, _, _ = struct.unpack("!HHHHHH", reply[:12])
        assert ident == 0xB0D1 and flags & 15 == 0 and answers > 0, "DNS lookup failed"


def verify_invalid_recovery(lab):
    original = '{"log":{"level":"debug"},"inbounds":[{"type":"not-a-protocol"}]}'
    _, client, proxy, root = lab.start("invalid", original)
    assert client.ok("/api/config/setup")["config_error"]
    request = {"reset_invalid": True, "inbound_mode": "proxy"}
    preview = client.ok("/api/config/setup/preview", "POST", request)
    assert preview["current_config"] == json.loads(original), "reset preview hides original values"
    assert "config_reset" in preview["warnings"]
    request["source_hash"] = preview["source_hash"]
    client.ok("/api/config/setup/apply", "POST", request)
    backup = root / "data/config-backups" / ("before-setup-" + preview["source_hash"] + ".json")
    assert backup.read_text() == original and backup.stat().st_mode & 0o777 == 0o600
    assert not client.ok("/api/service/status")["running"]
    client.ok("/api/service/start", "POST")
    handshake(proxy)


def verify_health(name):
    command("podman", "healthcheck", "run", name)
    state = json.loads(command("podman", "inspect", name))[0]["State"]
    health = state.get("Health") or state.get("Healthcheck") or {}
    assert health.get("Status") == "healthy", "container health check failed"


def verify_ipv4_tun(lab):
    extra = ["--cap-add", "NET_ADMIN", "--device", "/dev/net/tun",
             "--sysctl", "net.ipv6.conf.all.disable_ipv6=1",
             "--sysctl", "net.ipv6.conf.default.disable_ipv6=1"]
    name, client, _, _ = lab.start("ipv4-tun", extra=extra)
    client.ok("/api/service/stop", "POST")
    request = {"modules": ["inbounds"], "inbound_mode": "tun", "ipv6": "auto"}
    preview = client.ok("/api/config/setup/preview", "POST", request)
    tun = next(item for item in preview["config"]["inbounds"] if item["type"] == "tun")
    assert all(":" not in address for address in tun["address"])
    request["source_hash"] = preview["source_hash"]
    client.ok("/api/config/setup/apply", "POST", request)
    assert not client.ok("/api/service/status")["running"]
    client.ok("/api/service/start", "POST")
    assert client.ok("/api/service/status")["running"]
    command("podman", "exec", name, "ip", "link", "show", "boxd0")


def run(image):
    command("podman", "image", "exists", image)
    health = json.loads(command("podman", "image", "inspect", image))[0]["Config"].get("Healthcheck")
    assert health and health.get("Test"), "build the image with --format docker to preserve health checks"
    with tempfile.TemporaryDirectory(prefix="boxd-container-smoke-") as directory:
        lab = Lab(image, directory)
        try:
            name, client, proxy, root = lab.start("fresh")
            verify_first_start(client, proxy, root)
            verify_health(name)
            _, empty, empty_proxy, empty_root = lab.start("empty", "{}\n")
            verify_first_start(empty, empty_proxy, empty_root)
            node = make_node(lab)
            verify_import_and_proxy(client, proxy, node)
            verify_dns(proxy)
            verify_failed_apply(client, proxy)
            verify_stale_preview(client)
            before = hashlib.sha256((root / "config/config.json").read_bytes()).hexdigest()
            command("podman", "restart", name)
            client = Client(command("podman", "port", name, "9091/tcp"), lab.password)
            assert client.ok("/api/service/status")["running"]
            assert hashlib.sha256((root / "config/config.json").read_bytes()).hexdigest() == before
            handshake(proxy)
            verify_health(name)
            verify_invalid_recovery(lab)
            verify_ipv4_tun(lab)
            print("PASS: startup/recovery, health, proxy HTTPS/DNS, stopped import, rollback, stale preview, restart, IPv4-only TUN")
        finally:
            lab.close()


if __name__ == "__main__":
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("image", help="local boxd image built with --format docker")
    run(parser.parse_args().image)
