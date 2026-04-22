const { makeApp, registerUser, request } = require('./helpers');

describe('auth', () => {
  test('register creates a user and returns token', async () => {
    const { app } = makeApp();
    const res = await request(app)
      .post('/api/auth/register')
      .send({ username: 'alice', password: 'secret1', display_name: 'Alice' }); // pragma: allowlist secret
    expect(res.status).toBe(201);
    expect(res.body.token).toBeDefined();
    expect(res.body.user.username).toBe('alice');
  });

  test('register rejects short username', async () => {
    const { app } = makeApp();
    const res = await request(app)
      .post('/api/auth/register')
      .send({ username: 'a', password: 'secret1' }); // pragma: allowlist secret
    expect(res.status).toBe(400);
  });

  test('register rejects duplicate username', async () => {
    const { app } = makeApp();
    await registerUser(app, 'bob');
    const res = await request(app)
      .post('/api/auth/register')
      .send({ username: 'bob', password: 'secret1' }); // pragma: allowlist secret
    expect(res.status).toBe(409);
  });

  test('login succeeds with valid credentials', async () => {
    const { app } = makeApp();
    await registerUser(app, 'carol', 'pw12345'); // pragma: allowlist secret
    const res = await request(app)
      .post('/api/auth/login')
      .send({ username: 'carol', password: 'pw12345' }); // pragma: allowlist secret
    expect(res.status).toBe(200);
    expect(res.body.token).toBeDefined();
  });

  test('login fails with wrong password', async () => {
    const { app } = makeApp();
    await registerUser(app, 'dave', 'pw12345'); // pragma: allowlist secret
    const res = await request(app)
      .post('/api/auth/login')
      .send({ username: 'dave', password: 'wrong' }); // pragma: allowlist secret
    expect(res.status).toBe(401);
  });

  test('protected route requires token', async () => {
    const { app } = makeApp();
    const res = await request(app).get('/api/users/me');
    expect(res.status).toBe(401);
  });

  test('GET /api/users/me returns current user', async () => {
    const { app } = makeApp();
    const { token, user } = await registerUser(app, 'eve');
    const res = await request(app).get('/api/users/me').set('Authorization', `Bearer ${token}`);
    expect(res.status).toBe(200);
    expect(res.body.id).toBe(user.id);
    expect(res.body.username).toBe('eve');
  });
});
