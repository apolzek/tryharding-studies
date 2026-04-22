const axios = require('axios');
const fs = require('fs');
const FormData = require('form-data');
const tokens = require('./tokens');
const { cfg } = require('../config');

const NAME = 'facebook';
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
        'pages_show_list',
        'pages_manage_posts',
        'pages_read_engagement',
        'publish_video',
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
    const pages = await axios.get(`${GRAPH()}/me/accounts`, {
      params: { access_token: tokRes.data.access_token },
    });
    const page = pages.data.data?.[0];
    if (!page) throw new Error('No Facebook Page under this account. Create one first.');
    tokens.save(NAME, { access_token: page.access_token, page_id: page.id, page_name: page.name });
  },

  async postVideo({ title, description, filePath }) {
    const t = tokens.load(NAME);
    if (!t) throw new Error('Not authenticated with Facebook');

    const form = new FormData();
    form.append('title', title);
    form.append('description', description || '');
    form.append('access_token', t.access_token);
    form.append('source', fs.createReadStream(filePath));

    const r = await axios.post(`${GRAPH()}/${t.page_id}/videos`, form, {
      headers: form.getHeaders(),
      maxBodyLength: Infinity,
      maxContentLength: Infinity,
    });
    return { id: r.data.id, page: t.page_name };
  },
};
