"""RedQueen config loader — reads YAML, overlays env vars."""
from __future__ import annotations

import os
from functools import lru_cache
from pathlib import Path
from typing import Any

import yaml
from pydantic import BaseModel, Field


class OllamaCfg(BaseModel):
    base_url: str = "http://localhost:11434"
    model: str = "llama3.2"
    stream: bool = True
    keep_alive: str = "5m"
    timeout: int = 120


class LLMCfg(BaseModel):
    provider: str = "ollama"
    ollama: OllamaCfg = Field(default_factory=OllamaCfg)
    system_prompt: str = "You are RedQueen."
    max_history: int = 20
    temperature: float = 0.7


class BrowserVoiceCfg(BaseModel):
    stt_lang: str = "en-US"
    tts_voice: str = ""
    tts_rate: float = 1.0
    tts_pitch: float = 0.6


class VoiceCfg(BaseModel):
    enabled: bool = True
    browser: BrowserVoiceCfg = Field(default_factory=BrowserVoiceCfg)
    wake_word: str = "red queen"


class DiscordVoiceCfg(BaseModel):
    enabled: bool = True
    auto_join: bool = False
    idle_timeout: int = 300


class DiscordCfg(BaseModel):
    enabled: bool = False
    token: str = ""
    command_prefix: str = "!rq"
    voice: DiscordVoiceCfg = Field(default_factory=DiscordVoiceCfg)
    allowed_guilds: list[int] = Field(default_factory=list)
    channel_whitelist: list[int] = Field(default_factory=list)


class ThemeCfg(BaseModel):
    primary: str = "#ff0033"
    secondary: str = "#ff5566"
    background: str = "#05000a"
    accent: str = "#8b0000"
    text: str = "#ff9aa8"


class EffectsCfg(BaseModel):
    matrix_rain: bool = True
    glitch: bool = True
    scanlines: bool = True
    crt_flicker: bool = True
    boot_sequence: bool = True


class UICfg(BaseModel):
    title: str = "RED QUEEN // HIVE MAINFRAME"
    theme: ThemeCfg = Field(default_factory=ThemeCfg)
    effects: EffectsCfg = Field(default_factory=EffectsCfg)
    ascii_art: str = "red_queen"
    greeting: str = "Welcome, operator."


class ServerCfg(BaseModel):
    host: str = "0.0.0.0"
    port: int = 8000
    log_level: str = "info"
    cors_origins: list[str] = Field(default_factory=lambda: ["*"])


class Config(BaseModel):
    server: ServerCfg = Field(default_factory=ServerCfg)
    llm: LLMCfg = Field(default_factory=LLMCfg)
    voice: VoiceCfg = Field(default_factory=VoiceCfg)
    discord: DiscordCfg = Field(default_factory=DiscordCfg)
    ui: UICfg = Field(default_factory=UICfg)


def _apply_env_overrides(data: dict[str, Any]) -> dict[str, Any]:
    token = os.getenv("REDQUEEN_DISCORD_TOKEN")
    if token:
        data.setdefault("discord", {})["token"] = token
    ollama_url = os.getenv("REDQUEEN_OLLAMA_URL")
    if ollama_url:
        data.setdefault("llm", {}).setdefault("ollama", {})["base_url"] = ollama_url
    ollama_model = os.getenv("REDQUEEN_OLLAMA_MODEL")
    if ollama_model:
        data.setdefault("llm", {}).setdefault("ollama", {})["model"] = ollama_model
    return data


def load_config(path: str | Path | None = None) -> Config:
    cfg_path = Path(path or os.getenv("REDQUEEN_CONFIG", "config.yaml"))
    if not cfg_path.is_absolute():
        cfg_path = (Path(__file__).resolve().parent.parent / cfg_path).resolve()
    if not cfg_path.exists():
        return Config()
    raw = yaml.safe_load(cfg_path.read_text()) or {}
    raw = _apply_env_overrides(raw)
    return Config.model_validate(raw)


@lru_cache(maxsize=1)
def get_config() -> Config:
    return load_config()


def reload_config() -> Config:
    get_config.cache_clear()
    return get_config()
