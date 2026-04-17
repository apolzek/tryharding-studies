"""Discord integration for RedQueen.

Text commands respond via LLM. Voice channel support: joins the caller's
channel and speaks responses using TTS (requires ffmpeg + gTTS or similar
downstream tooling). Kept minimal here — the hook points are marked.
"""
from __future__ import annotations

import logging
from typing import TYPE_CHECKING

import discord
from discord.ext import commands

if TYPE_CHECKING:
    from .config import Config
    from .llm import OllamaClient

log = logging.getLogger("redqueen.discord")


def build_bot(cfg: "Config", llm: "OllamaClient") -> commands.Bot:
    intents = discord.Intents.default()
    intents.message_content = True
    intents.voice_states = True
    bot = commands.Bot(command_prefix=cfg.discord.command_prefix + " ", intents=intents)

    history: dict[int, list[dict[str, str]]] = {}

    def _allowed(guild_id: int | None, channel_id: int | None) -> bool:
        if cfg.discord.allowed_guilds and guild_id not in cfg.discord.allowed_guilds:
            return False
        if cfg.discord.channel_whitelist and channel_id not in cfg.discord.channel_whitelist:
            return False
        return True

    @bot.event
    async def on_ready():
        log.info("RedQueen online as %s", bot.user)

    @bot.command(name="ask")
    async def ask(ctx: commands.Context, *, prompt: str):
        if not _allowed(ctx.guild.id if ctx.guild else None, ctx.channel.id):
            return
        chan_hist = history.setdefault(ctx.channel.id, [])
        async with ctx.typing():
            reply = await llm.chat(chan_hist, prompt)
        chan_hist.append({"role": "user", "content": prompt})
        chan_hist.append({"role": "assistant", "content": reply})
        for chunk in _chunks(reply, 1900):
            await ctx.send(chunk)

    @bot.command(name="join")
    async def join(ctx: commands.Context):
        if not cfg.discord.voice.enabled:
            await ctx.send("[voice module disabled]")
            return
        if not ctx.author.voice or not ctx.author.voice.channel:
            await ctx.send("[operator is not in a voice channel]")
            return
        await ctx.author.voice.channel.connect()
        await ctx.send("[RedQueen has entered the hive]")

    @bot.command(name="leave")
    async def leave(ctx: commands.Context):
        if ctx.voice_client:
            await ctx.voice_client.disconnect(force=False)
            await ctx.send("[RedQueen disengaged]")

    @bot.command(name="reset")
    async def reset(ctx: commands.Context):
        history.pop(ctx.channel.id, None)
        await ctx.send("[memory purged]")

    return bot


def _chunks(s: str, n: int):
    for i in range(0, len(s), n):
        yield s[i : i + n]


async def start_discord(cfg: "Config", llm: "OllamaClient") -> None:
    bot = build_bot(cfg, llm)
    await bot.start(cfg.discord.token)
