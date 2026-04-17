"""Config loader tests."""
from __future__ import annotations

import os
from pathlib import Path

import yaml

from backend.config import Config, load_config


def test_load_default_yaml():
    cfg = load_config()
    assert isinstance(cfg, Config)
    assert cfg.server.port == 8000
    assert cfg.llm.provider == "ollama"
    assert cfg.llm.ollama.model
    assert cfg.ui.title


def test_env_overrides(tmp_path, monkeypatch):
    yml = tmp_path / "c.yaml"
    yml.write_text(
        yaml.safe_dump(
            {
                "llm": {"ollama": {"base_url": "http://x:1", "model": "m-1"}},
                "discord": {"enabled": True, "token": "original"},
            }
        )
    )
    monkeypatch.setenv("REDQUEEN_DISCORD_TOKEN", "from-env")
    monkeypatch.setenv("REDQUEEN_OLLAMA_URL", "http://env-host:11434")
    monkeypatch.setenv("REDQUEEN_OLLAMA_MODEL", "env-model")
    cfg = load_config(yml)
    assert cfg.discord.token == "from-env"
    assert cfg.llm.ollama.base_url == "http://env-host:11434"
    assert cfg.llm.ollama.model == "env-model"


def test_missing_file_returns_defaults(tmp_path):
    cfg = load_config(tmp_path / "nope.yaml")
    assert isinstance(cfg, Config)
    assert cfg.llm.ollama.base_url.startswith("http")


def test_theme_fields_present():
    cfg = load_config()
    t = cfg.ui.theme
    assert t.primary.startswith("#") and len(t.primary) == 7
    assert t.background.startswith("#")
