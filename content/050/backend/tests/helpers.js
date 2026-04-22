const request = require('supertest');
const { createApp } = require('../src/app');
const { openDb, migrate } = require('../src/db');

function makeApp() {
  const db = openDb(':memory:');
  migrate(db);
  const app = createApp(db);
  return { app, db };
}

async function registerUser(app, username, password = 'secret1', display_name) {
  const res = await request(app)
    .post('/api/auth/register')
    .send({ username, password, display_name: display_name || username });
  return res.body; // { token, user }
}

function authHeader(token) {
  return { Authorization: `Bearer ${token}` };
}

module.exports = { makeApp, registerUser, authHeader, request };
