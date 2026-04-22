# 046 — Multi-post Video

Publish one video to **YouTube, TikTok, Instagram Reels, Facebook, X (Twitter) and LinkedIn** from one UI. Twitch is included for auth only — Twitch has no public VOD upload API.

```
046/
├── server.js            # Express backend
├── config.js            # Runtime config (env + config.json), masked reads
├── services/            # One module per platform (auth + upload)
│   ├── youtube.js       # googleapis, resumable upload
│   ├── tiktok.js        # Content Posting API (inbox or direct)
│   ├── twitch.js        # auth only, throws on postVideo
│   ├── instagram.js     # Graph API Reels container flow
│   ├── facebook.js      # Graph API /{page-id}/videos
│   ├── twitter.js       # api.x.com/2/media/upload (v1.1 sunset 2025-06-09)
│   └── linkedin.js      # /rest/videos + /rest/posts (v2/assets deprecated)
├── public/              # Vanilla-JS B&W 3D frontend
│   ├── index.html       # Tabs: Publish / Connect / Settings
│   ├── app.js
│   └── style.css
├── tokens/              # OAuth tokens per platform (gitignored)
├── uploads/             # Temp storage for uploaded videos (gitignored)
├── public-media/        # Served publicly so Instagram can fetch videos
├── config.json          # Runtime credentials (gitignored)
└── .env.example
```

## Quick start

```bash
cd content/046
npm install
npm start
# open http://localhost:3000
```

No `.env` required to start — everything can be configured from the **Settings** tab in the UI. If you do set env vars (via `.env` or the shell), they take precedence over `config.json`.

## UI — three tabs

- **Publish** — pick a video, fill title/description/tags, tick target platforms (checkboxes stay disabled until that platform is both connected and supports uploading), click POST. Results show per-platform ok/error with IDs and URLs.
- **Connect** — one card per platform with live status (connected / not connected), Connect/Reconnect buttons and a Disconnect button that drops the token file.
- **Settings** — form-based editor for every credential, grouped by platform. Each group shows the exact callback URL you have to whitelist at the provider. Secrets are masked: the server sends a `xxx…yyy` hint but never the full value back to the browser; leave a secret field blank to keep what's stored.

The same form can be used to change `APP_URL`, the Meta Graph version (`META_GRAPH_VERSION`, default `v22.0`), TikTok mode (`inbox` vs `direct`), etc.

## How each platform is integrated (verified April 2026)

| Platform  | Auth                    | Upload path                                                               |
| --------- | ----------------------- | ------------------------------------------------------------------------- |
| YouTube   | OAuth 2.0 (web)         | `googleapis` resumable upload (`videos.insert`)                           |
| TikTok    | OAuth 2.0               | Content Posting API — `publish/video/init` **or** `inbox/video/init` + PUT|
| Twitch    | OAuth 2.0               | **No upload API** — auth only, POST throws a friendly error               |
| Instagram | Meta OAuth              | Graph API `v22.0` Reels container + `media_publish` (needs PUBLIC_URL)    |
| Facebook  | Meta OAuth              | Graph API `v22.0` `/{page-id}/videos` multipart                           |
| X         | OAuth 1.0a user ctx     | `api.x.com/2/media/upload` chunked + `POST /2/tweets` (v1.1 was sunset 2025-06-09) |
| LinkedIn  | OAuth 2.0 + OIDC        | `/rest/videos?action=initializeUpload` + multi-part PUT + `finalizeUpload` + `/rest/posts`, header `LinkedIn-Version: 202604` |

The old LinkedIn `/v2/assets?action=registerUpload` + `/v2/ugcPosts` flow is deprecated (replaced by the Videos API + Posts API). The Twitter `upload.twitter.com/1.1/media/upload.json` endpoint was fully sunset on **2025-06-09**. This project uses the current endpoints.

## Credential sources

Every value below can be entered in the **Settings** tab (saved to `config.json`) OR exported as an env var. Env wins if both are set.

### General

| Key          | What it does                                                                 |
| ------------ | ---------------------------------------------------------------------------- |
| `APP_URL`    | Base URL for OAuth redirect URIs. Local dev: `http://localhost:3000`.        |
| `PUBLIC_URL` | Public HTTPS URL (e.g. ngrok) — required for Instagram so Meta can fetch videos. |

### YouTube

1. <https://console.cloud.google.com/> → **APIs & Services** → enable **YouTube Data API v3**.
2. **Credentials** → **Create Credentials** → **OAuth client ID** → *Web application*.
3. Authorized redirect URI: `${APP_URL}/api/auth/youtube/callback`.
4. Add your Google account as a **Test user** under the OAuth consent screen (otherwise refresh tokens expire after 7 days).

Keys: `YOUTUBE_CLIENT_ID`, `YOUTUBE_CLIENT_SECRET`, `YOUTUBE_PRIVACY` (private | unlisted | public).

### TikTok

1. <https://developers.tiktok.com/> → create an app, enable **Login Kit** + **Content Posting API**.
2. Redirect URI: `${APP_URL}/api/auth/tiktok/callback`.
3. Choose a mode via `TIKTOK_MODE`:
   - `inbox` (default) — video is sent to the creator's TikTok app drafts; human finishes publishing. Only needs `video.upload` scope. Safer while your app is unaudited.
   - `direct` — publishes immediately. Needs `video.publish` scope. Unaudited apps are forced to `privacy_level=SELF_ONLY` (the service pre-queries `creator_info` and falls back automatically).

Keys: `TIKTOK_CLIENT_KEY`, `TIKTOK_CLIENT_SECRET`, `TIKTOK_MODE`, `TIKTOK_PRIVACY`.

### Twitch (auth only)

Twitch removed the public VOD upload endpoint. Auth is kept for completeness. Register at <https://dev.twitch.tv/console/apps>; redirect URL `${APP_URL}/api/auth/twitch/callback`.

Keys: `TWITCH_CLIENT_ID`, `TWITCH_CLIENT_SECRET`.

### Instagram + Facebook (same Meta app)

IG publishing needs a **Business / Creator** Instagram account linked to a **Facebook Page**. (The newer "Instagram API with Instagram Login" path is not used here — this flow works for Business accounts managed through a Page.)

1. <https://developers.facebook.com/apps/> → create app (type *Business*).
2. Add **Facebook Login for Business** and **Instagram Graph API** products.
3. Valid OAuth Redirect URIs (both):
   - `${APP_URL}/api/auth/facebook/callback`
   - `${APP_URL}/api/auth/instagram/callback`
4. Request permissions:
   - `pages_show_list`, `pages_manage_posts`, `pages_read_engagement`, `publish_video`
   - `instagram_basic`, `instagram_content_publish`, `business_management`
5. Add yourself as a **Tester** in App Roles while developing.

Keys: `META_APP_ID`, `META_APP_SECRET`, `META_GRAPH_VERSION` (default `v22.0`).

### X (Twitter)

X's video upload requires **OAuth 1.0a User Context**. Rather than implementing a user-facing OAuth 1.0a flow we use the credentials of *your own* developer account (single-user app).

1. <https://developer.x.com/> → create a Project + App with **Read and write** permissions.
2. In **Keys and tokens**: generate API Key/Secret **and** Access Token/Secret for your own account. Generate the access tokens *after* setting write permission, otherwise regenerate them.
3. Any paid tier works for posting; Free is also allowed with strict monthly write limits.

Keys: `TWITTER_API_KEY`, `TWITTER_API_SECRET`, `TWITTER_ACCESS_TOKEN`, `TWITTER_ACCESS_SECRET`.

### LinkedIn

1. <https://developer.linkedin.com/> → create an app.
2. Products to request:
   - **Sign In with LinkedIn using OpenID Connect** (for `openid profile email`)
   - **Share on LinkedIn** (for `w_member_social`)
3. In **Auth** tab → authorized redirect URL: `${APP_URL}/api/auth/linkedin/callback`.

Keys: `LINKEDIN_CLIENT_ID`, `LINKEDIN_CLIENT_SECRET`.

## Testing end-to-end

1. `npm install && npm start`.
2. Open <http://localhost:3000> → **Settings** tab → fill credentials for whatever platforms you want → **Save**.
3. **Connect** tab → click *Connect* on each platform. For X you don't connect — credentials are read straight from settings.
4. **Publish** tab → pick a short MP4 (10–30s is plenty), fill title/description, tick targets, click **POST**.
5. Results show per-platform `ok` / `error` with IDs and URLs.

## Known gotchas

- **YouTube**: new projects in *Testing* mode expire refresh tokens after 7 days → either publish the consent screen or reconnect weekly.
- **TikTok**: `inbox` mode is the safest default — posts land as drafts in the user's TikTok app. `direct` mode requires the Content Posting API audit; unaudited apps can only post `SELF_ONLY`.
- **Instagram**: needs a Business/Creator IG linked to a FB Page, and `PUBLIC_URL` must be reachable from the internet (use ngrok during dev).
- **Meta**: pin the Graph version you tested against (`META_GRAPH_VERSION`) — Meta deprecates versions roughly every 2 years.
- **X**: 512 MB / 140 s for standard accounts; longer videos need an X Premium / verified posting user.
- **LinkedIn**: `w_member_social` posts on behalf of the authenticated person only. Company-page posting requires `w_organization_social` (Marketing Developer Platform).
- **Twitch**: no public upload API — the checkbox auto-disables.

## Architecture

Each platform is `services/<platform>.js`:

```js
{
  authFlow: 'oauth' | 'static',
  supportsUpload?: boolean,        // default true; Twitch sets false
  isAuthenticated(): boolean,
  getAuthUrl?(): string,           // omit for static-auth platforms
  handleCallback?(req): Promise,   // omit for static-auth platforms
  postVideo({ title, description, tags, filePath, mimeType }): Promise<object>,
}
```

`server.js` auto-wires every service into `/api/auth/:platform`, `/api/auth/:platform/callback`, `/api/auth/:platform/disconnect`, `/api/auth/status`, `/api/settings` (GET/POST) and `/api/upload`. Adding a new platform = one new file in `services/` + one entry in the `platforms` map in `server.js`.
