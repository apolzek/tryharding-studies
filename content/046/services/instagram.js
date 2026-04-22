const axios = require('axios');
const path = require('path');
const fs = require('fs');
const tokens = require('./tokens');
const { cfg } = require('../config');

const NAME = 'instagram';
const V = () => cfg('META_GRAPH_VERSION') || 'v22.0';
const AUTHORIZE_URL = () => `https://www.facebook.com/${V()}/dialog/oauth`;
const TOKEN_URL = () => `https://graph.facebook.com/${V()}/oauth/access_token`;
const GRAPH = () => `https://graph.facebook.com/${V()}`;
const REDIRECT = () => `${cfg('APP_URL')}/api/auth/${NAME}/callback`;

module.exports = {
  authFlow: 'oauth',
  isAuthenticated: () => !!tokens.load(NAME),

  getAuthUrl() {
    if (!cfg('META_APP_ID')) throw new Error('META_APP_ID missing');
    const params = new URLSearchParams({
      client_id: cfg('META_APP_ID'),
      redirect_uri: REDIRECT(),
      scope: [
        'instagram_basic',
        'instagram_content_publish',
        'pages_show_list',
        'pages_read_engagement',
        'business_management',
      ].join(','),
      response_type: 'code',
    });
    return `${AUTHORIZE_URL()}?${params}`;
  },

  async handleCallback(req) {
    const tokRes = await axios.get(TOKEN_URL(), {
      params: {
        client_id: cfg('META_APP_ID'),
        client_secret: cfg('META_APP_SECRET'),
        redirect_uri: REDIRECT(),
        code: req.query.code,
      },
    });
    const userToken = tokRes.data.access_token;

    const pages = await axios.get(`${GRAPH()}/me/accounts`, { params: { access_token: userToken } });
    const page = pages.data.data?.[0];
    if (!page) throw new Error('No Facebook Page linked to this account. Create a Page first.');

    const ig = await axios.get(`${GRAPH()}/${page.id}`, {
      params: { fields: 'instagram_business_account', access_token: page.access_token },
    });

    tokens.save(NAME, {
      access_token: page.access_token,
      page_id: page.id,
      ig_user_id: ig.data.instagram_business_account?.id || null,
    });
  },

  async postVideo({ title, description, filePath }) {
    const t = tokens.load(NAME);
    if (!t) throw new Error('Not authenticated with Instagram');
    if (!t.ig_user_id) throw new Error('No Instagram Business/Creator account linked to your Facebook Page');
    if (!cfg('PUBLIC_URL')) {
      throw new Error('PUBLIC_URL required for Instagram (IG fetches the video from it). Expose via ngrok and set it in Settings.');
    }

    const filename = `${Date.now()}-${path.basename(filePath)}.mp4`;
    const publicFile = path.join(__dirname, '..', 'public-media', filename);
    fs.copyFileSync(filePath, publicFile);
    const videoUrl = `${cfg('PUBLIC_URL').replace(/\/$/, '')}/public-media/${filename}`;

    try {
      const container = await axios.post(`${GRAPH()}/${t.ig_user_id}/media`, null, {
        params: {
          media_type: 'REELS',
          video_url: videoUrl,
          caption: [title, description].filter(Boolean).join('\n\n'),
          access_token: t.access_token,
        },
      });
      const creationId = container.data.id;

      let finished = false;
      for (let i = 0; i < 40; i++) {
        await new Promise(r => setTimeout(r, 4000));
        const s = await axios.get(`${GRAPH()}/${creationId}`, {
          params: { fields: 'status_code', access_token: t.access_token },
        });
        if (s.data.status_code === 'FINISHED') { finished = true; break; }
        if (s.data.status_code === 'ERROR') throw new Error('Instagram container transitioned to ERROR');
      }
      if (!finished) throw new Error('Instagram container still not FINISHED after polling');

      const pub = await axios.post(`${GRAPH()}/${t.ig_user_id}/media_publish`, null, {
        params: { creation_id: creationId, access_token: t.access_token },
      });

      return { id: pub.data.id };
    } finally {
      setTimeout(() => fs.unlink(publicFile, () => {}), 60_000);
    }
  },
};
