const axios = require('axios');
const fs = require('fs');
const tokens = require('./tokens');
const { cfg } = require('../config');

const NAME = 'linkedin';
const AUTHORIZE_URL = 'https://www.linkedin.com/oauth/v2/authorization';
const TOKEN_URL = 'https://www.linkedin.com/oauth/v2/accessToken';
const VERSION = '202604';
const REDIRECT = () => `${cfg('APP_URL')}/api/auth/${NAME}/callback`;

function restHeaders(accessToken) {
  return {
    Authorization: `Bearer ${accessToken}`,
    'LinkedIn-Version': VERSION,
    'X-Restli-Protocol-Version': '2.0.0',
    'Content-Type': 'application/json',
  };
}

module.exports = {
  authFlow: 'oauth',
  isAuthenticated: () => !!tokens.load(NAME),

  getAuthUrl() {
    if (!cfg('LINKEDIN_CLIENT_ID')) throw new Error('LINKEDIN_CLIENT_ID missing');
    const params = new URLSearchParams({
      response_type: 'code',
      client_id: cfg('LINKEDIN_CLIENT_ID'),
      redirect_uri: REDIRECT(),
      scope: 'openid profile email w_member_social',
      state: 'li-' + Date.now(),
    });
    return `${AUTHORIZE_URL}?${params}`;
  },

  async handleCallback(req) {
    const tokRes = await axios.post(TOKEN_URL, new URLSearchParams({
      grant_type: 'authorization_code',
      code: req.query.code,
      redirect_uri: REDIRECT(),
      client_id: cfg('LINKEDIN_CLIENT_ID'),
      client_secret: cfg('LINKEDIN_CLIENT_SECRET'),
    }), { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } });

    const accessToken = tokRes.data.access_token;
    const ui = await axios.get('https://api.linkedin.com/v2/userinfo', {
      headers: { Authorization: `Bearer ${accessToken}` },
    });

    tokens.save(NAME, {
      access_token: accessToken,
      person_urn: `urn:li:person:${ui.data.sub}`,
      name: ui.data.name,
      obtained_at: Date.now(),
    });
  },

  async postVideo({ title, description, filePath }) {
    const t = tokens.load(NAME);
    if (!t) throw new Error('Not authenticated with LinkedIn');

    const size = fs.statSync(filePath).size;

    const init = await axios.post(
      'https://api.linkedin.com/rest/videos?action=initializeUpload',
      {
        initializeUploadRequest: {
          owner: t.person_urn,
          fileSizeBytes: size,
          uploadCaptions: false,
          uploadThumbnail: false,
        },
      },
      { headers: restHeaders(t.access_token) },
    );

    const { value } = init.data;
    const videoUrn = value.video;
    const uploadToken = value.uploadToken;
    const instructions = value.uploadInstructions;

    const fd = fs.openSync(filePath, 'r');
    const partIds = [];
    try {
      for (const inst of instructions) {
        const { firstByte, lastByte, uploadUrl } = inst;
        const len = lastByte - firstByte + 1;
        const buf = Buffer.alloc(len);
        fs.readSync(fd, buf, 0, len, firstByte);
        const put = await axios.put(uploadUrl, buf, {
          headers: { 'Content-Type': 'application/octet-stream' },
          maxBodyLength: Infinity,
          maxContentLength: Infinity,
        });
        const etag = put.headers['etag'] || put.headers['ETag'];
        if (!etag) throw new Error('No ETag returned by LinkedIn for a part');
        partIds.push(etag.replace(/^"|"$/g, ''));
      }
    } finally {
      fs.closeSync(fd);
    }

    await axios.post(
      'https://api.linkedin.com/rest/videos?action=finalizeUpload',
      {
        finalizeUploadRequest: {
          video: videoUrn,
          uploadToken,
          uploadedPartIds: partIds,
        },
      },
      { headers: restHeaders(t.access_token) },
    );

    let available = false;
    for (let i = 0; i < 30; i++) {
      await new Promise(r => setTimeout(r, 3000));
      const encoded = encodeURIComponent(videoUrn);
      const s = await axios.get(`https://api.linkedin.com/rest/videos/${encoded}`, {
        headers: restHeaders(t.access_token),
      });
      if (s.data.status === 'AVAILABLE') { available = true; break; }
      if (s.data.status === 'PROCESSING_FAILED') throw new Error('LinkedIn video processing failed');
    }
    if (!available) throw new Error('LinkedIn video still not AVAILABLE after polling');

    const post = await axios.post(
      'https://api.linkedin.com/rest/posts',
      {
        author: t.person_urn,
        commentary: [title, description].filter(Boolean).join('\n\n'),
        visibility: 'PUBLIC',
        distribution: {
          feedDistribution: 'MAIN_FEED',
          targetEntities: [],
          thirdPartyDistributionChannels: [],
        },
        content: { media: { title, id: videoUrn } },
        lifecycleState: 'PUBLISHED',
        isReshareDisabledByAuthor: false,
      },
      { headers: restHeaders(t.access_token) },
    );

    const postId = post.headers['x-restli-id'] || post.data.id || 'unknown';
    return { id: postId, video: videoUrn };
  },
};
