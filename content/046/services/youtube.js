const { google } = require('googleapis');
const fs = require('fs');
const tokens = require('./tokens');
const { cfg } = require('../config');

const NAME = 'youtube';
const SCOPES = ['https://www.googleapis.com/auth/youtube.upload'];

function oauthClient() {
  return new google.auth.OAuth2(
    cfg('YOUTUBE_CLIENT_ID'),
    cfg('YOUTUBE_CLIENT_SECRET'),
    `${cfg('APP_URL')}/api/auth/${NAME}/callback`,
  );
}

module.exports = {
  authFlow: 'oauth',
  isAuthenticated: () => !!tokens.load(NAME),

  getAuthUrl() {
    if (!cfg('YOUTUBE_CLIENT_ID')) throw new Error('YOUTUBE_CLIENT_ID missing');
    return oauthClient().generateAuthUrl({
      access_type: 'offline',
      scope: SCOPES,
      prompt: 'consent',
    });
  },

  async handleCallback(req) {
    const client = oauthClient();
    const { tokens: t } = await client.getToken(req.query.code);
    tokens.save(NAME, t);
  },

  async postVideo({ title, description, tags, filePath }) {
    const t = tokens.load(NAME);
    if (!t) throw new Error('Not authenticated with YouTube');
    const client = oauthClient();
    client.setCredentials(t);
    const yt = google.youtube({ version: 'v3', auth: client });
    const res = await yt.videos.insert({
      part: ['snippet', 'status'],
      requestBody: {
        snippet: { title, description, tags },
        status: {
          privacyStatus: cfg('YOUTUBE_PRIVACY') || 'private',
          selfDeclaredMadeForKids: false,
        },
      },
      media: { body: fs.createReadStream(filePath) },
    });
    const id = res.data.id;
    return { id, url: `https://youtu.be/${id}` };
  },
};
