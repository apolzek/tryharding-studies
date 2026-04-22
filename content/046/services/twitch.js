const axios = require('axios');
const tokens = require('./tokens');
const { cfg } = require('../config');

const NAME = 'twitch';
const AUTHORIZE_URL = 'https://id.twitch.tv/oauth2/authorize';
const TOKEN_URL = 'https://id.twitch.tv/oauth2/token';
const REDIRECT = () => `${cfg('APP_URL')}/api/auth/${NAME}/callback`;

module.exports = {
  authFlow: 'oauth',
  supportsUpload: false,

  isAuthenticated: () => !!tokens.load(NAME),

  getAuthUrl() {
    if (!cfg('TWITCH_CLIENT_ID')) throw new Error('TWITCH_CLIENT_ID missing');
    const params = new URLSearchParams({
      client_id: cfg('TWITCH_CLIENT_ID'),
      redirect_uri: REDIRECT(),
      response_type: 'code',
      scope: 'user:read:email channel:manage:videos',
    });
    return `${AUTHORIZE_URL}?${params}`;
  },

  async handleCallback(req) {
    const r = await axios.post(TOKEN_URL, new URLSearchParams({
      client_id: cfg('TWITCH_CLIENT_ID'),
      client_secret: cfg('TWITCH_CLIENT_SECRET'),
      code: req.query.code,
      grant_type: 'authorization_code',
      redirect_uri: REDIRECT(),
    }), { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } });
    tokens.save(NAME, { ...r.data, obtained_at: Date.now() });
  },

  async postVideo() {
    throw new Error(
      'Twitch does not expose a public API to upload a VOD. ' +
      'Use Twitch Studio / OBS / Video Producer to upload VODs.',
    );
  },
};
