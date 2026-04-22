const { makeApp, registerUser, authHeader, request } = require('./helpers');

describe('users', () => {
  test('update profile via PUT /me', async () => {
    const { app } = makeApp();
    const { token } = await registerUser(app, 'frank');
    const res = await request(app)
      .put('/api/users/me')
      .set(authHeader(token))
      .send({ bio: 'hello world', status: 'namorando', age: 28, city: 'Sampa' });
    expect(res.status).toBe(200);
    expect(res.body.bio).toBe('hello world');
    expect(res.body.status).toBe('namorando');
    expect(res.body.age).toBe(28);
  });

  test('search users by query', async () => {
    const { app } = makeApp();
    const { token } = await registerUser(app, 'gina');
    await registerUser(app, 'gabriel');
    await registerUser(app, 'zeca');
    const res = await request(app).get('/api/users/search?q=g').set(authHeader(token));
    expect(res.status).toBe(200);
    const names = res.body.map((u) => u.username).sort();
    expect(names).toEqual(expect.arrayContaining(['gabriel', 'gina']));
    expect(names).not.toContain('zeca');
  });

  test('GET /api/users/:id returns public fields', async () => {
    const { app } = makeApp();
    const { token } = await registerUser(app, 'henry');
    const { user } = await registerUser(app, 'ivy');
    const res = await request(app).get(`/api/users/${user.id}`).set(authHeader(token));
    expect(res.status).toBe(200);
    expect(res.body.username).toBe('ivy');
    expect(res.body.password_hash).toBeUndefined();
  });
});
