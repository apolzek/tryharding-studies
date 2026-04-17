import http from 'node:http';
import { WebSocketServer } from 'ws';

const PORT = 8004;
const server = http.createServer((req, res) => {
  if (req.url === '/health') {
    res.writeHead(200);
    return res.end('ok');
  }
  res.writeHead(404);
  res.end();
});

const wss = new WebSocketServer({ server, path: '/ws' });

function broadcast(data) {
  for (const client of wss.clients) {
    if (client.readyState === 1) client.send(data);
  }
}

wss.on('connection', (ws, req) => {
  ws.isAlive = true;
  console.log(`[js-ws] client connected (total=${wss.clients.size}) from=${req.socket.remoteAddress}`);
  ws.on('pong', () => { ws.isAlive = true; });
  ws.on('message', (data) => {
    const msg = data.toString();
    console.log(`[js-ws] received: ${msg}`);
    broadcast(msg);
  });
  ws.on('close', () => console.log(`[js-ws] client disconnected (total=${wss.clients.size})`));
});

const interval = setInterval(() => {
  for (const ws of wss.clients) {
    if (!ws.isAlive) { ws.terminate(); continue; }
    ws.isAlive = false;
    ws.ping();
  }
}, 30000);

wss.on('close', () => clearInterval(interval));

server.listen(PORT, () => console.log(`[js-ws] listening on :${PORT}`));
