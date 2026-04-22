const axios = require('axios');
const crypto = require('crypto');
const fs = require('fs');
const FormData = require('form-data');
const OAuth = require('oauth-1.0a');
const { cfg } = require('../config');

const UPLOAD_URL = 'https://api.x.com/2/media/upload';
const TWEET_URL = 'https://api.x.com/2/tweets';

function creds() {
  return {
    apiKey: cfg('TWITTER_API_KEY'),
    apiSecret: cfg('TWITTER_API_SECRET'),
    accessToken: cfg('TWITTER_ACCESS_TOKEN'),
    accessSecret: cfg('TWITTER_ACCESS_SECRET'),
  };
}

function oauthClient() {
  const { apiKey, apiSecret } = creds();
  return OAuth({
    consumer: { key: apiKey, secret: apiSecret },
    signature_method: 'HMAC-SHA1',
    hash_function: (base, key) =>
      crypto.createHmac('sha1', key).update(base).digest('base64'),
  });
}

function authHeader(url, method, params) {
  const { accessToken, accessSecret } = creds();
  const oauth = oauthClient();
  return oauth.toHeader(
    oauth.authorize(
      { url, method, data: params },
      { key: accessToken, secret: accessSecret },
    ),
  );
}

module.exports = {
  authFlow: 'static',

  isAuthenticated() {
    const c = creds();
    return !!(c.apiKey && c.apiSecret && c.accessToken && c.accessSecret);
  },

  async postVideo({ title, description, filePath }) {
    if (!this.isAuthenticated()) {
      throw new Error('Twitter credentials missing. Set all four TWITTER_* keys in Settings or .env');
    }

    const size = fs.statSync(filePath).size;

    const initParams = {
      command: 'INIT',
      total_bytes: String(size),
      media_type: 'video/mp4',
      media_category: 'tweet_video',
    };
    const initHeaders = authHeader(UPLOAD_URL, 'POST', initParams);
    const init = await axios.post(UPLOAD_URL, new URLSearchParams(initParams), {
      headers: { ...initHeaders, 'Content-Type': 'application/x-www-form-urlencoded' },
    });
    const data = init.data.data || init.data;
    const mediaId = data.id || data.media_id_string || data.media_id;
    if (!mediaId) throw new Error('X INIT did not return a media id: ' + JSON.stringify(init.data));

    const CHUNK = 4 * 1024 * 1024;
    const fd = fs.openSync(filePath, 'r');
    try {
      const buf = Buffer.alloc(CHUNK);
      let pos = 0, idx = 0;
      while (pos < size) {
        const n = fs.readSync(fd, buf, 0, CHUNK, pos);
        const form = new FormData();
        form.append('command', 'APPEND');
        form.append('media_id', String(mediaId));
        form.append('segment_index', String(idx));
        form.append('media', buf.slice(0, n), {
          filename: 'chunk.bin',
          contentType: 'application/octet-stream',
        });
        const hdr = authHeader(UPLOAD_URL, 'POST', {});
        await axios.post(UPLOAD_URL, form, {
          headers: { ...hdr, ...form.getHeaders() },
          maxBodyLength: Infinity,
          maxContentLength: Infinity,
        });
        pos += n;
        idx += 1;
      }
    } finally {
      fs.closeSync(fd);
    }

    const finParams = { command: 'FINALIZE', media_id: String(mediaId) };
    const finHeaders = authHeader(UPLOAD_URL, 'POST', finParams);
    const fin = await axios.post(UPLOAD_URL, new URLSearchParams(finParams), {
      headers: { ...finHeaders, 'Content-Type': 'application/x-www-form-urlencoded' },
    });
    const finData = fin.data.data || fin.data;

    let info = finData.processing_info;
    while (info && info.state !== 'succeeded') {
      if (info.state === 'failed') {
        throw new Error('X media processing failed: ' + JSON.stringify(info));
      }
      await new Promise(r => setTimeout(r, (info.check_after_secs || 2) * 1000));
      const params = { command: 'STATUS', media_id: String(mediaId) };
      const hdr = authHeader(UPLOAD_URL, 'GET', params);
      const s = await axios.get(UPLOAD_URL, { params, headers: hdr });
      const sd = s.data.data || s.data;
      info = sd.processing_info;
    }

    const tweetText = [title, description].filter(Boolean).join('\n\n').slice(0, 280);
    const tweetHdr = authHeader(TWEET_URL, 'POST', {});
    const tweet = await axios.post(
      TWEET_URL,
      { text: tweetText, media: { media_ids: [String(mediaId)] } },
      { headers: { ...tweetHdr, 'Content-Type': 'application/json' } },
    );

    const id = tweet.data.data.id;
    return { id, url: `https://x.com/i/web/status/${id}` };
  },
};
