const { makeApp, registerUser, authHeader, request } = require('./helpers');

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

describe('visits', () => {
  test('records visitor and groups by latest visit', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'visA');
    const b = await registerUser(app, 'visB');
    const c = await registerUser(app, 'visC');

    await request(app).post(`/api/visits/${a.user.id}`).set(authHeader(b.token));
    await sleep(5);
    await request(app).post(`/api/visits/${a.user.id}`).set(authHeader(c.token));
    await sleep(5);
    await request(app).post(`/api/visits/${a.user.id}`).set(authHeader(b.token));

    const list = await request(app).get(`/api/visits/${a.user.id}`).set(authHeader(a.token));
    expect(list.body).toHaveLength(2);
    expect(list.body[0].username).toBe('visB');
  });

  test('self-visit is skipped', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'svA');
    const res = await request(app).post(`/api/visits/${a.user.id}`).set(authHeader(a.token));
    expect(res.body.skipped).toBe(true);
  });
});
