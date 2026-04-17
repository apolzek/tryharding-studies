"""Static asset sanity checks — no browser runtime."""
from __future__ import annotations

from pathlib import Path

FRONTEND = Path(__file__).resolve().parent.parent / "frontend"


def test_index_html_present_and_links_assets():
    html = (FRONTEND / "index.html").read_text()
    assert 'href="/static/style.css"' in html
    assert '/static/app.js' in html
    assert '/static/ascii.js' in html
    assert 'id="prompt"' in html
    assert 'id="console"' in html
    assert 'id="mic"' in html
    assert 'id="matrix"' in html


def test_app_js_has_ws_and_voice():
    js = (FRONTEND / "app.js").read_text()
    assert "WebSocket" in js
    assert "SpeechRecognition" in js
    assert "speechSynthesis" in js
    assert "matrix" in js.lower()


def test_ascii_js_has_art():
    js = (FRONTEND / "ascii.js").read_text()
    assert "REDQUEEN_ASCII" in js
    assert "red_queen" in js
    assert "REDQUEEN_BOOT" in js


def test_style_has_glitch_and_scanlines():
    css = (FRONTEND / "style.css").read_text()
    assert ".glitch" in css
    assert "#scanlines" in css
    assert "@keyframes" in css
