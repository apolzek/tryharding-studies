const { makeApp, registerUser, authHeader, request } = require('./helpers');

describe('communities', () => {
  test('create, list, join, leave', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'coA');
    const b = await registerUser(app, 'coB');

    const created = await request(app)
      .post('/api/communities')
      .set(authHeader(a.token))
      .send({ name: 'Eu odeio acordar cedo', description: 'classico', category: 'Humor' });
    expect(created.status).toBe(201);
    expect(created.body.name).toBe('Eu odeio acordar cedo');

    const mineA = await request(app).get('/api/communities/mine').set(authHeader(a.token));
    expect(mineA.body).toHaveLength(1);

    const join = await request(app)
      .post(`/api/communities/${created.body.id}/join`)
      .set(authHeader(b.token));
    expect(join.status).toBe(200);

    const detail = await request(app).get(`/api/communities/${created.body.id}`).set(authHeader(b.token));
    expect(detail.body.member_count).toBe(2);

    const leave = await request(app)
      .post(`/api/communities/${created.body.id}/leave`)
      .set(authHeader(b.token));
    expect(leave.body.left).toBe(1);
  });

  test('search communities by query', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'searchA');
    await request(app).post('/api/communities').set(authHeader(a.token)).send({ name: 'Bandas Rock' });
    await request(app).post('/api/communities').set(authHeader(a.token)).send({ name: 'Amo Gatos' });
    const res = await request(app).get('/api/communities?q=rock').set(authHeader(a.token));
    expect(res.body).toHaveLength(1);
    expect(res.body[0].name).toBe('Bandas Rock');
  });

  test('cannot join twice', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'joinA');
    const c = await request(app).post('/api/communities').set(authHeader(a.token)).send({ name: 'X' });
    const dup = await request(app).post(`/api/communities/${c.body.id}/join`).set(authHeader(a.token));
    expect(dup.status).toBe(409);
  });
});
