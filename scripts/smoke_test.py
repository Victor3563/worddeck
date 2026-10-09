#!/usr/bin/env python3
"""Сквозная проверка REST API WordDeck.

Запуск: python3 scripts/smoke_test.py [BASE_URL] [--rate-limit]
        BASE_URL по умолчанию http://localhost:8080 (nginx из docker compose).
        Тест делает 9 запросов входа и регистрации — в пределах NGINX_AUTH_BURST (20).
        --rate-limit дополнительно проверяет ответ 429; после него перед следующим
        запуском нужно подождать минуту.
"""
import http.cookiejar
import json
import sys
import time
import urllib.error
import urllib.request
import uuid

ARGS = [a for a in sys.argv[1:] if not a.startswith("--")]
BASE = (ARGS[0] if ARGS else "http://localhost:8080").rstrip("/")
CHECK_RATE_LIMIT = "--rate-limit" in sys.argv
COOKIE = "wd_session"
passed = 0


class Client:

    def __init__(self):
        self.jar = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar))

    def cookie(self):
        return next((c for c in self.jar if c.name == COOKIE), None)


ANON = Client()


def call(method, path, body=None, user=ANON, raw=None, headers=None, expect=None):
    global passed
    hdrs = dict(headers or {})
    data = None
    if raw is not None:
        data = raw
    elif body is not None:
        data = json.dumps(body).encode()
        hdrs["Content-Type"] = "application/json"
    req = urllib.request.Request(BASE + path, data=data, method=method, headers=hdrs)
    try:
        with user.opener.open(req, timeout=15) as r:
            status, payload, ctype, resp_headers = r.status, r.read(), r.headers.get("Content-Type", ""), r.headers
    except urllib.error.HTTPError as e:
        status, payload, ctype, resp_headers = e.code, e.read(), e.headers.get("Content-Type", ""), e.headers
    parsed = json.loads(payload) if payload and "json" in ctype else payload
    if expect is not None and status != expect:
        raise AssertionError(f"{method} {path}: expected {expect}, got {status}: {parsed}")
    passed += 1
    call.last_headers = resp_headers
    return status, parsed


def error_code(parsed):
    return parsed["error"]["code"]


def multipart(fields, files):
    boundary = uuid.uuid4().hex
    out = b""
    for k, v in fields.items():
        out += f'--{boundary}\r\nContent-Disposition: form-data; name="{k}"\r\n\r\n{v}\r\n'.encode()
    for k, (name, content) in files.items():
        out += (f'--{boundary}\r\nContent-Disposition: form-data; name="{k}"; filename="{name}"\r\n'
                f"Content-Type: text/plain\r\n\r\n").encode() + content + b"\r\n"
    out += f"--{boundary}--\r\n".encode()
    return out, {"Content-Type": f"multipart/form-data; boundary={boundary}"}


def main():
    suffix = str(int(time.time() * 1000))
    email_a, email_b, pwd = f"alice{suffix}@example.com", f"bob{suffix}@example.com", "password123"

    call("GET", "/api/health-that-does-not-exist", expect=404)

    ta, tb, ta2 = Client(), Client(), Client()  # Алиса, Боб, Алиса на втором устройстве
    _, err = call("POST", "/api/auth/register", {"email": email_a, "display_name": "A", "password": "short"}, expect=422)
    assert err["error"] == {"code": "too_short", "field": "password", "params": {"min": 8}}, err
    _, a = call("POST", "/api/auth/register", {"email": email_a, "display_name": "Алиса", "password": pwd}, user=ta, expect=201)
    assert "token" not in a and a["user"]["email"] == email_a, "the session must not be in the response body"
    cookie = ta.cookie()
    assert cookie and cookie.path == "/api" and cookie.has_nonstandard_attr("HttpOnly"), cookie
    set_cookie = call.last_headers.get("Set-Cookie", "")
    assert "SameSite=Strict" in set_cookie and "HttpOnly" in set_cookie, set_cookie
    _, err = call("POST", "/api/auth/register", {"email": email_a.upper(), "display_name": "X", "password": pwd}, expect=422)
    assert error_code(err) == "taken"
    call("POST", "/api/auth/register", {"email": email_b, "display_name": "Боб", "password": pwd}, user=tb, expect=201)
    _, err = call("POST", "/api/auth/login", {"email": email_a, "password": "wrong-password"}, expect=401)
    assert error_code(err) == "invalid_credentials"
    call("POST", "/api/auth/login", {"email": email_a, "password": pwd}, user=ta2, expect=200)
    call("GET", "/api/me", expect=401)
    call("GET", "/api/me", headers={"Cookie": COOKIE + "=forged"}, expect=401)
    _, me = call("GET", "/api/me", user=ta, expect=200)
    assert me["display_name"] == "Алиса"
    call("GET", "/api/me", user=ta2, expect=200)
    call("PATCH", "/api/me", {"display_name": "Алиса Л."}, user=ta, expect=200)

    _, wss = call("GET", "/api/workspaces", user=ta, expect=200)
    assert len(wss) == 1, "default workspace must be created on registration"
    call("POST", "/api/workspaces", {"name": "  "}, user=ta, expect=422)
    _, ws = call("POST", "/api/workspaces", {"name": "Английский B2", "description": "Слова к экзамену", "color": "indigo"}, user=ta, expect=201)
    call("POST", "/api/workspaces", {"name": "x", "color": "neon"}, user=ta, expect=422)
    _, ws = call("PATCH", f"/api/workspaces/{ws['id']}", {"description": "Обновлено"}, user=ta, expect=200)
    assert ws["name"] == "Английский B2" and ws["description"] == "Обновлено"
    call("GET", f"/api/workspaces/{ws['id']}", user=tb, expect=404)  # чужой ресурс не виден
    call("DELETE", f"/api/workspaces/{ws['id']}", user=tb, expect=404)

    _, f1 = call("POST", f"/api/workspaces/{ws['id']}/folders", {"name": "Дом"}, user=ta, expect=201)
    _, f2 = call("POST", f"/api/workspaces/{ws['id']}/folders", {"name": "Еда", "description": "Кухня"}, user=ta, expect=201)
    call("POST", f"/api/workspaces/{ws['id']}/folders", {"name": "Чужая"}, user=tb, expect=404)
    _, folders = call("GET", f"/api/workspaces/{ws['id']}/folders", user=ta, expect=200)
    assert [f["name"] for f in folders] == ["Дом", "Еда"]
    call("PATCH", f"/api/folders/{f2['id']}", {"name": "Еда и напитки"}, user=ta, expect=200)

    call("POST", f"/api/folders/{f1['id']}/cards", {"ru_word": "", "en_word": "x"}, user=ta, expect=422)
    _, c1 = call("POST", f"/api/folders/{f1['id']}/cards",
                 {"ru_word": "дом", "ru_example": "Я иду домой.", "en_word": "house", "en_example": "I'm going home."}, user=ta, expect=201)
    _, bulk = call("POST", f"/api/folders/{f1['id']}/cards/bulk",
                   {"cards": [{"ru_word": "окно", "en_word": "window"}, {"ru_word": "дверь", "en_word": "door"}]}, user=ta, expect=201)
    assert len(bulk) == 2
    _, cards = call("GET", f"/api/folders/{f1['id']}/cards", user=ta, expect=200)
    assert len(cards) == 3
    _, found = call("GET", f"/api/folders/{f1['id']}/cards?q=WIN", user=ta, expect=200)
    assert len(found) == 1 and found[0]["en_word"] == "window"
    _, c1 = call("PATCH", f"/api/cards/{c1['id']}", {"en_example": "Home, sweet home."}, user=ta, expect=200)
    assert c1["en_example"] == "Home, sweet home." and c1["ru_word"] == "дом"
    call("PATCH", f"/api/cards/{c1['id']}", {"ru_word": " "}, user=ta, expect=422)
    call("GET", f"/api/cards/{c1['id']}", user=tb, expect=404)
    call("PATCH", f"/api/cards/{bulk[1]['id']}", {"folder_id": f2["id"]}, user=ta, expect=200)
    _, moved = call("GET", f"/api/folders/{f2['id']}/cards", user=ta, expect=200)
    assert len(moved) == 1

    text = "ru_word;ru_example;en_word;en_example\nкот;Кот спит.;cat;The cat sleeps.\nсобака;;dog;\nплохая строка\n".encode()
    body, hdrs = multipart({}, {"file": ("words.csv", text)})
    _, preview = call("POST", f"/api/folders/{f2['id']}/import?dry_run=true", raw=body, headers=hdrs, user=ta, expect=200)
    assert preview["parsed"] == 2 and len(preview["errors"]) == 1 and len(preview["cards"]) == 2
    body, hdrs = multipart({}, {"file": ("words.csv", text)})
    _, imp = call("POST", f"/api/folders/{f2['id']}/import", raw=body, headers=hdrs, user=ta, expect=201)
    assert imp["created"] == 2
    body, hdrs = multipart({"text": "солнце\tsun\nлуна\tmoon"}, {})
    _, imp = call("POST", f"/api/folders/{f2['id']}/import", raw=body, headers=hdrs, user=ta, expect=201)
    assert imp["created"] == 2

    _, csv_body = call("GET", f"/api/folders/{f2['id']}/export?format=csv", user=ta, expect=200)
    assert b"cat" in csv_body and b"sun" in csv_body
    _, js = call("GET", f"/api/folders/{f2['id']}/export?format=json", user=ta, expect=200)
    assert len(js["cards"]) == 5

    _, all_cards = call("GET", f"/api/workspaces/{ws['id']}/cards", user=ta, expect=200)
    assert len(all_cards) == 7

    _, share = call("POST", "/api/shares", {"resource_type": "folder", "resource_id": f2["id"]}, user=ta, expect=201)
    _, share2 = call("POST", "/api/shares", {"resource_type": "folder", "resource_id": f2["id"]}, user=ta, expect=201)
    assert share["token"] == share2["token"], "sharing twice must return the same link"
    call("POST", "/api/shares", {"resource_type": "folder", "resource_id": f2["id"]}, user=tb, expect=404)
    _, pub = call("GET", f"/api/public/shares/{share['token']}", expect=200)
    assert pub["resource_type"] == "folder" and len(pub["folder"]["cards"]) == 5 and not pub["is_owner"]
    _, pub = call("GET", f"/api/public/shares/{share['token']}", user=ta, expect=200)
    assert pub["is_owner"]
    _, ws_share = call("POST", "/api/shares", {"resource_type": "workspace", "resource_id": ws["id"]}, user=ta, expect=201)
    _, card_share = call("POST", "/api/shares", {"resource_type": "card", "resource_id": c1["id"]}, user=ta, expect=201)
    _, shares = call("GET", "/api/shares", user=ta, expect=200)
    assert len(shares) == 3
    _, filtered = call("GET", f"/api/shares?type=card&id={c1['id']}", user=ta, expect=200)
    assert len(filtered) == 1 and filtered[0]["resource_name"] == "house — дом"

    _, bob_ws = call("GET", "/api/workspaces", user=tb, expect=200)
    call("POST", f"/api/shares/{share['token']}/copy", {}, user=tb, expect=422)  # для папки нужна целевая область
    call("POST", f"/api/shares/{share['token']}/copy", {"target_id": ws["id"]}, user=tb, expect=404)  # не его область
    _, cp = call("POST", f"/api/shares/{share['token']}/copy", {"target_id": bob_ws[0]["id"]}, user=tb, expect=201)
    assert cp["cards_copied"] == 5
    _, cp = call("POST", f"/api/shares/{ws_share['token']}/copy", {}, user=tb, expect=201)
    assert cp["cards_copied"] == 7
    _, cp_card = call("POST", f"/api/shares/{card_share['token']}/copy", {"target_id": cp_folder_id(tb, bob_ws[0]["id"])}, user=tb, expect=201)
    _, bob_ws = call("GET", "/api/workspaces", user=tb, expect=200)
    assert len(bob_ws) == 2 and bob_ws[0]["card_count"] == 6

    call("DELETE", f"/api/shares/{card_share['token']}", user=tb, expect=404)
    call("DELETE", f"/api/shares/{card_share['token']}", user=ta, expect=204)
    call("GET", f"/api/public/shares/{card_share['token']}", expect=404)

    call("POST", "/api/study-sessions", {"mode": "fly", "total_cards": 1}, user=ta, expect=422)
    for mode in ("learn", "dictation", "match"):
        call("POST", "/api/study-sessions", {"mode": mode, "folder_id": f1["id"], "title": "Дом",
                                               "total_cards": 3, "mistakes": 1, "rounds": 2, "duration_ms": 65000}, user=ta, expect=201)
    _, sessions = call("GET", "/api/study-sessions?limit=2", user=ta, expect=200)
    assert sessions["total"] == 3 and len(sessions["items"]) == 2
    call("DELETE", f"/api/study-sessions/{sessions['items'][0]['id']}", user=tb, expect=404)
    call("DELETE", f"/api/study-sessions/{sessions['items'][0]['id']}", user=ta, expect=204)
    _, stats = call("GET", "/api/stats?tz=Europe/Moscow", user=ta, expect=200)
    assert stats["sessions"] == 2 and stats["streak_days"] == 1 and stats["cards"] == 7 and len(stats["last_days"]) == 14

    call("POST", f"/api/folders/{f2['id']}/cards/delete", {"ids": [moved[0]["id"]]}, user=ta, expect=200)
    call("DELETE", f"/api/cards/{c1['id']}", user=ta, expect=204)
    call("DELETE", f"/api/cards/{c1['id']}", user=ta, expect=404)
    call("DELETE", f"/api/folders/{f1['id']}", user=ta, expect=204)
    call("DELETE", f"/api/workspaces/{ws['id']}", user=ta, expect=204)
    call("GET", f"/api/public/shares/{share['token']}", expect=404)  # ссылка удалена вместе с папкой

    _, err = call("PUT", "/api/me/password", {"current_password": "nope", "new_password": "newpassword1"}, user=ta, expect=422)
    assert error_code(err) == "wrong_password"
    call("PUT", "/api/me/password", {"current_password": pwd, "new_password": "newpassword1"}, user=ta, expect=204)
    call("GET", "/api/me", user=ta, expect=200)  # текущая сессия переживает смену пароля
    call("GET", "/api/me", user=ta2, expect=401)  # остальные сессии завершены
    call("POST", "/api/auth/login", {"email": email_a, "password": "newpassword1"}, user=ta2, expect=200)
    call("POST", "/api/auth/logout-all", user=ta, expect=204)
    call("GET", "/api/me", user=ta, expect=401)
    call("GET", "/api/me", user=ta2, expect=401)
    tb2 = Client()
    call("POST", "/api/auth/login", {"email": email_b, "password": pwd}, user=tb2, expect=200)
    call("POST", "/api/auth/logout", user=tb2, expect=204)
    call("GET", "/api/me", user=tb2, expect=401)
    call("GET", "/api/me", user=tb, expect=200)  # logout завершает только свою сессию
    call("DELETE", "/api/me", {"password": pwd}, user=tb, expect=204)
    call("GET", "/api/me", user=tb, expect=401)  # удаление аккаунта завершает его сессии
    call("POST", "/api/auth/login", {"email": email_a, "password": "newpassword1"}, user=ta, expect=200)
    call("DELETE", "/api/me", {"password": "newpassword1"}, user=ta, expect=204)

    if CHECK_RATE_LIMIT:
        for _ in range(30):
            status, parsed = call("POST", "/api/auth/login", {"email": email_b, "password": "x" * 8})
            if status == 429:
                assert error_code(parsed) == "rate_limited", parsed
                break
        else:
            raise AssertionError("login attempts were not rate-limited")

    print(f"OK: {passed} API checks passed against {BASE}")


def cp_folder_id(user, workspace_id):
    _, folders = call("GET", f"/api/workspaces/{workspace_id}/folders", user=user, expect=200)
    return folders[0]["id"]


if __name__ == "__main__":
    try:
        main()
    except AssertionError as e:
        print("FAIL:", e)
        sys.exit(1)
