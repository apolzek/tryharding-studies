const { createApp } = require('./app');
const { openDb, migrate } = require('./db');

const PORT = Number(process.env.PORT || 4000);
const DB_PATH = process.env.DB_PATH || './data/orkut.db';

const db = openDb(DB_PATH);
migrate(db);

const app = createApp(db);

app.listen(PORT, () => {
  console.log(`[orkut-backend] listening on :${PORT}`);
});
