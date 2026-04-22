const { makeApp, registerUser, authHeader, request } = require('./helpers');

describe('ratings', () => {
  test('upsert rating and summary reflects averages', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'raterA');
    const b = await registerUser(app, 'raterB');
    const c = await registerUser(app, 'rateeC');

    await request(app).put(`/api/ratings/${c.user.id}`).set(authHeader(a.token))
      .send({ trust: 3, cool: 2, sexy: 1, is_fan: true });
    await request(app).put(`/api/ratings/${c.user.id}`).set(authHeader(b.token))
      .send({ trust: 1, cool: 2, sexy: 3, is_fan: false });

    const sum = await request(app).get(`/api/ratings/${c.user.id}/summary`).set(authHeader(a.token));
    expect(sum.body.summary.raters).toBe(2);
    expect(sum.body.summary.fans).toBe(1);
    expect(sum.body.summary.trust).toBe(2);
    expect(sum.body.summary.cool).toBe(2);
    expect(sum.body.summary.sexy).toBe(2);
    expect(sum.body.mine).toEqual({ trust: 3, cool: 2, sexy: 1, is_fan: 1 });
  });

  test('cannot rate yourself', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'self');
    const res = await request(app).put(`/api/ratings/${a.user.id}`).set(authHeader(a.token))
      .send({ trust: 3 });
    expect(res.status).toBe(400);
  });

  test('values are clamped to 0..3', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'clampA');
    const b = await registerUser(app, 'clampB');
    const res = await request(app).put(`/api/ratings/${b.user.id}`).set(authHeader(a.token))
      .send({ trust: 99, cool: -5, sexy: 2 });
    expect(res.body.trust).toBe(3);
    expect(res.body.cool).toBe(0);
    expect(res.body.sexy).toBe(2);
  });

  test('fans endpoint lists users who marked as fan', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'fanA');
    const b = await registerUser(app, 'fanB');
    const c = await registerUser(app, 'fanC');
    await request(app).put(`/api/ratings/${c.user.id}`).set(authHeader(a.token)).send({ is_fan: true });
    await request(app).put(`/api/ratings/${c.user.id}`).set(authHeader(b.token)).send({ is_fan: true });
    const res = await request(app).get(`/api/ratings/${c.user.id}/fans`).set(authHeader(c.token));
    expect(res.body.map((u) => u.username).sort()).toEqual(['fanA', 'fanB']);
  });
});
