import http from 'node:http';
import crypto from 'node:crypto';

const PORT = 9004;
const SECRET = process.env.WEBHOOK_SECRET || 's3cret';
let received = 0;

function verifyHmac(body, sig) {
  if (!sig) return true;
  const expected = crypto.createHmac('sha256', SECRET).update(body).digest('hex');
  try {
    return crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(sig));
  } catch { return false; }
}

const server = http.createServer(async (req, res) => {
  if (req.method === 'GET' && req.url === '/health') {
    res.writeHead(200); return res.end('ok');
  }
  if (req.method === 'GET' && req.url === '/stats') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    return res.end(JSON.stringify({ received, ts: Math.floor(Date.now() / 1000) }));
  }
  if (req.method === 'POST' && req.url === '/webhook') {
    const chunks = [];
    for await (const c of req) chunks.push(c);
    const body = Buffer.concat(chunks);
    if (!verifyHmac(body, req.headers['x-signature'])) {
      res.writeHead(401); return res.end('invalid signature');
    }
    let payload = {};
    try { payload = JSON.parse(body.toString()); } catch {}
    received += 1;
    console.log(`[js-wh] #${received} event=${payload.event}`);
    res.writeHead(202, { 'Content-Type': 'application/json' });
    return res.end(JSON.stringify({ status: 'accepted' }));
  }
  res.writeHead(404); res.end();
});

server.listen(PORT, () => console.log(`[js-wh] listening on :${PORT}`));
