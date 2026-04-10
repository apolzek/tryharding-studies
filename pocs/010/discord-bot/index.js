require('dotenv').config();
const {
  Client,
  GatewayIntentBits,
  Events,
} = require('discord.js');
const {
  joinVoiceChannel,
  EndBehaviorType,
  VoiceConnectionStatus,
  entersState,
} = require('@discordjs/voice');
const fs = require('fs');
const path = require('path');
const https = require('https');
const http = require('http');
const { pipeline } = require('stream');

const WHISPER_API_URL = process.env.WHISPER_API_URL || 'http://localhost:8001/transcribe';

const TARGET_CHANNEL_NAME = 'robo';
const AUDIO_OUTPUT_DIR = path.join(__dirname, 'recordings');
const TEXT_LOG_FILE = path.join(__dirname, 'messages.log');

if (!fs.existsSync(AUDIO_OUTPUT_DIR)) {
  fs.mkdirSync(AUDIO_OUTPUT_DIR, { recursive: true });
}

const client = new Client({
  intents: [
    GatewayIntentBits.Guilds,
    GatewayIntentBits.GuildMessages,
    GatewayIntentBits.MessageContent,
    GatewayIntentBits.GuildVoiceStates,
  ],
});

// ─── AUDIO TRANSCRIPTION ──────────────────────────────────────────────────────

function downloadFile(url, destPath) {
  return new Promise((resolve, reject) => {
    const protocol = url.startsWith('https') ? https : http;
    const file = fs.createWriteStream(destPath);
    protocol.get(url, (res) => {
      res.pipe(file);
      file.on('finish', () => { file.close(); resolve(); });
    }).on('error', (err) => {
      fs.unlink(destPath, () => {});
      reject(err);
    });
  });
}

async function transcribeAttachment(message, att) {
  const tmpPath = path.join(AUDIO_OUTPUT_DIR, `transcribe_${Date.now()}_${att.name}`);
  try {
    await downloadFile(att.url, tmpPath);

    const boundary = `----FormBoundary${Date.now()}`;
    const fileData = fs.readFileSync(tmpPath);
    const body = Buffer.concat([
      Buffer.from(
        `--${boundary}\r\nContent-Disposition: form-data; name="file"; filename="${att.name}"\r\nContent-Type: application/octet-stream\r\n\r\n`
      ),
      fileData,
      Buffer.from(`\r\n--${boundary}--\r\n`),
    ]);

    const result = await new Promise((resolve, reject) => {
      const url = new URL(WHISPER_API_URL);
      const options = {
        hostname: url.hostname,
        port: url.port || 80,
        path: url.pathname,
        method: 'POST',
        headers: {
          'Content-Type': `multipart/form-data; boundary=${boundary}`,
          'Content-Length': body.length,
        },
      };
      const req = http.request(options, (res) => {
        let data = '';
        res.on('data', (chunk) => { data += chunk; });
        res.on('end', () => {
          try { resolve(JSON.parse(data)); }
          catch (e) { reject(new Error(`Resposta inválida: ${data}`)); }
        });
      });
      req.on('error', reject);
      req.write(body);
      req.end();
    });

    const text = result.text ? result.text.trim() : '(sem fala detectada)';
    console.log(`[TRANSCRIPTION] ${message.author.tag}: ${text}`);
    const logEntry = `[${new Date().toISOString()}] TRANSCRIPTION | ${message.author.tag}: ${text}\n`;
    fs.appendFileSync(TEXT_LOG_FILE, logEntry);
    await message.reply(`**Transcrição de ${message.author.displayName}:** ${text}`);
  } catch (err) {
    console.error(`[TRANSCRIPTION] Erro ao transcrever ${att.name}:`, err.message);
  } finally {
    if (fs.existsSync(tmpPath)) fs.unlinkSync(tmpPath);
  }
}

// ─── TEXT MESSAGES ────────────────────────────────────────────────────────────

client.on(Events.MessageCreate, async (message) => {
  if (message.author.bot) return;
  if (message.channel.name !== TARGET_CHANNEL_NAME) return;

  const entry = `[${new Date().toISOString()}] ${message.guild.name} | #${message.channel.name} | ${message.author.tag}: ${message.content}\n`;
  console.log('[TEXT]', entry.trim());
  fs.appendFileSync(TEXT_LOG_FILE, entry);

  if (message.attachments.size > 0) {
    for (const att of message.attachments.values()) {
      const contentType = att.contentType || '';
      console.log(`[ATTACHMENT] ${att.name} (${contentType}) -> ${att.url}`);
      const attEntry = `[${new Date().toISOString()}] ATTACHMENT: ${att.name} | ${att.url}\n`;
      fs.appendFileSync(TEXT_LOG_FILE, attEntry);

      if (contentType.startsWith('audio/')) {
        await transcribeAttachment(message, att);
      }
    }
  }
});

// ─── VOICE CHANNEL ────────────────────────────────────────────────────────────

const activeConnections = new Map();

async function joinAndRecord(voiceChannel) {
  if (activeConnections.has(voiceChannel.id)) return;

  console.log(`[VOICE] Entrando no canal de voz: ${voiceChannel.name}`);

  const connection = joinVoiceChannel({
    channelId: voiceChannel.id,
    guildId: voiceChannel.guild.id,
    adapterCreator: voiceChannel.guild.voiceAdapterCreator,
    selfDeaf: false,
    selfMute: true,
  });

  activeConnections.set(voiceChannel.id, connection);

  connection.on(VoiceConnectionStatus.Disconnected, async () => {
    try {
      await Promise.race([
        entersState(connection, VoiceConnectionStatus.Signalling, 5_000),
        entersState(connection, VoiceConnectionStatus.Connecting, 5_000),
      ]);
    } catch {
      connection.destroy();
      activeConnections.delete(voiceChannel.id);
      console.log(`[VOICE] Desconectado de ${voiceChannel.name}`);
    }
  });

  const receiver = connection.receiver;

  receiver.speaking.on('start', (userId) => {
    const user = client.users.cache.get(userId);
    const username = user ? user.tag : userId;
    console.log(`[VOICE] ${username} começou a falar`);

    const timestamp = Date.now();
    const filename = path.join(AUDIO_OUTPUT_DIR, `${username}_${timestamp}.pcm`);
    const fileStream = fs.createWriteStream(filename);

    const audioStream = receiver.subscribe(userId, {
      end: {
        behavior: EndBehaviorType.AfterSilence,
        duration: 1000,
      },
    });

    pipeline(audioStream, fileStream, (err) => {
      if (err) {
        console.error(`[VOICE] Erro ao gravar áudio de ${username}:`, err.message);
      } else {
        const stats = fs.statSync(filename);
        if (stats.size > 0) {
          console.log(`[VOICE] Áudio salvo: ${filename} (${stats.size} bytes)`);
        } else {
          fs.unlinkSync(filename); // Remove arquivos vazios
        }
      }
    });
  });

  receiver.speaking.on('end', (userId) => {
    const user = client.users.cache.get(userId);
    console.log(`[VOICE] ${user ? user.tag : userId} parou de falar`);
  });
}

// Detecta quando alguém entra em canal de voz chamado "robo"
client.on(Events.VoiceStateUpdate, async (oldState, newState) => {
  const guild = newState.guild;

  // Verifica se o canal destino existe e é de voz
  const targetVoice = guild.channels.cache.find(
    (c) => c.name === TARGET_CHANNEL_NAME && c.isVoiceBased()
  );

  if (!targetVoice) return;

  // Alguém entrou no canal alvo → bot entra também
  if (newState.channelId === targetVoice.id && !activeConnections.has(targetVoice.id)) {
    await joinAndRecord(targetVoice);
  }

  // Canal ficou vazio → bot sai
  const members = targetVoice.members.filter((m) => !m.user.bot);
  if (members.size === 0 && activeConnections.has(targetVoice.id)) {
    const conn = activeConnections.get(targetVoice.id);
    conn.destroy();
    activeConnections.delete(targetVoice.id);
    console.log(`[VOICE] Canal vazio, saindo de ${targetVoice.name}`);
  }
});

// ─── READY ────────────────────────────────────────────────────────────────────

client.once(Events.ClientReady, async (c) => {
  console.log(`[BOT] Online como ${c.user.tag}`);
  console.log(`[BOT] Monitorando canal: #${TARGET_CHANNEL_NAME}`);
  console.log(`[BOT] Textos → ${TEXT_LOG_FILE}`);
  console.log(`[BOT] Áudios → ${AUDIO_OUTPUT_DIR}/`);

  // Se já há alguém no canal de voz "robo" quando o bot liga
  for (const guild of c.guilds.cache.values()) {
    const voiceChannel = guild.channels.cache.find(
      (ch) => ch.name === TARGET_CHANNEL_NAME && ch.isVoiceBased()
    );
    if (voiceChannel) {
      const humans = voiceChannel.members.filter((m) => !m.user.bot);
      if (humans.size > 0) {
        await joinAndRecord(voiceChannel);
      }
    }
  }
});

client.login(process.env.DISCORD_TOKEN);
