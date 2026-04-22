const fs = require('fs');
const axios = require('axios');
const tokens = require('./tokens');
const { cfg } = require('../config');

const NAME = 'tiktok';
const AUTHORIZE_URL = 'https://www.tiktok.com/v2/auth/authorize/';
const TOKEN_URL = 'https://open.tiktokapis.com/v2/oauth/token/';
const CREATOR_INFO_URL = 'https://open.tiktokapis.com/v2/post/publish/creator_info/query/';
const DIRECT_INIT_URL = 'https://open.tiktokapis.com/v2/post/publish/video/init/';
const INBOX_INIT_URL = 'https://open.tiktokapis.com/v2/post/publish/inbox/video/init/';
const STATUS_URL = 'https://open.tiktokapis.com/v2/post/publish/status/fetch/';

const REDIRECT = () => `${cfg('APP_URL')}/api/auth/${NAME}/callback`;
const MODE = () => (cfg('TIKTOK_MODE') || 'inbox').toLowerCase();

module.exports = {
  authFlow: 'oauth',
  isAuthenticated: () => !!tokens.load(NAME),

  getAuthUrl() {
    if (!cfg('TIKTOK_CLIENT_KEY')) throw new Error('TIKTOK_CLIENT_KEY missing');
    const scopes = MODE() === 'direct'
      ? 'user.info.basic,video.publish,video.upload'
      : 'user.info.basic,video.upload';
    const params = new URLSearchParams({
      client_key: cfg('TIKTOK_CLIENT_KEY'),
      response_type: 'code',
      scope: scopes,
      redirect_uri: REDIRECT(),
      state: 'tk-' + Date.now(),
    });
    return `${AUTHORIZE_URL}?${params}`;
  },

  async handleCallback(req) {
    const r = await axios.post(TOKEN_URL, new URLSearchParams({
      client_key: cfg('TIKTOK_CLIENT_KEY'),
      client_secret: cfg('TIKTOK_CLIENT_SECRET'),
      code: req.query.code,
      grant_type: 'authorization_code',
      redirect_uri: REDIRECT(),
    }), { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } });
    tokens.save(NAME, { ...r.data, obtained_at: Date.now() });
  },

  async postVideo({ title, filePath }) {
    const t = tokens.load(NAME);
    if (!t) throw new Error('Not authenticated with TikTok');
    const size = fs.statSync(filePath).size;
    const isDirect = MODE() === 'direct';

    const initBody = {
      source_info: {
        source: 'FILE_UPLOAD',
        video_size: size,
        chunk_size: size,
        total_chunk_count: 1,
      },
    };

    if (isDirect) {
      let privacyLevel = cfg('TIKTOK_PRIVACY') || 'SELF_ONLY';
      try {
        const info = await axios.post(CREATOR_INFO_URL, {}, {
          headers: {
            Authorization: `Bearer ${t.access_token}`,
            'Content-Type': 'application/json; charset=UTF-8',
          },
        });
        const allowed = info.data?.data?.privacy_level_options || [];
        if (allowed.length && !allowed.includes(privacyLevel)) {
          privacyLevel = allowed[0];
        }
      } catch {
        /* creator_info only needed in direct mode; fall back to configured value */
      }
      initBody.post_info = {
        title: title.slice(0, 150),
        privacy_level: privacyLevel,
        disable_duet: false,
        disable_comment: false,
        disable_stitch: false,
      };
    }

    const initUrl = isDirect ? DIRECT_INIT_URL : INBOX_INIT_URL;
    const init = await axios.post(initUrl, initBody, {
      headers: {
        Authorization: `Bearer ${t.access_token}`,
        'Content-Type': 'application/json; charset=UTF-8',
      },
    });

    const { upload_url, publish_id } = init.data.data;

    const fileBuf = fs.readFileSync(filePath);
    await axios.put(upload_url, fileBuf, {
      headers: {
        'Content-Type': 'video/mp4',
        'Content-Length': size,
        'Content-Range': `bytes 0-${size - 1}/${size}`,
      },
      maxBodyLength: Infinity,
      maxContentLength: Infinity,
    });

    let status = 'PROCESSING';
    try {
      const s = await axios.post(STATUS_URL, { publish_id }, {
        headers: {
          Authorization: `Bearer ${t.access_token}`,
          'Content-Type': 'application/json; charset=UTF-8',
        },
      });
      status = s.data?.data?.status || status;
    } catch { /* ignore */ }

    return {
      publish_id,
      status,
      mode: isDirect ? 'direct' : 'inbox (open TikTok app to finish publishing)',
    };
  },
};
