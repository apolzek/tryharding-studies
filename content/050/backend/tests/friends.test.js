const { makeApp, registerUser, authHeader, request } = require('./helpers');

describe('friends', () => {
  test('request, pending list, accept, then list friends', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'frA');
    const b = await registerUser(app, 'frB');

    const reqRes = await request(app)
      .post(`/api/friends/request/${b.user.id}`)
      .set(authHeader(a.token));
    expect(reqRes.status).toBe(201);

    const pending = await request(app).get('/api/friends/pending').set(authHeader(b.token));
    expect(pending.body).toHaveLength(1);
    expect(pending.body[0].username).toBe('frA');

    const acc = await request(app)
      .post(`/api/friends/accept/${a.user.id}`)
      .set(authHeader(b.token));
    expect(acc.status).toBe(200);

    const listA = await request(app).get(`/api/friends/list/${a.user.id}`).set(authHeader(a.token));
    expect(listA.body.map((u) => u.username)).toEqual(['frB']);

    const listB = await request(app).get(`/api/friends/list/${b.user.id}`).set(authHeader(b.token));
    expect(listB.body.map((u) => u.username)).toEqual(['frA']);
  });

  test('cannot friend yourself', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'meself');
    const res = await request(app)
      .post(`/api/friends/request/${a.user.id}`)
      .set(authHeader(a.token));
    expect(res.status).toBe(400);
  });

  test('duplicate request returns 409', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'dupA');
    const b = await registerUser(app, 'dupB');
    await request(app).post(`/api/friends/request/${b.user.id}`).set(authHeader(a.token));
    const dup = await request(app).post(`/api/friends/request/${b.user.id}`).set(authHeader(a.token));
    expect(dup.status).toBe(409);
  });

  test('remove friendship', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'rmA');
    const b = await registerUser(app, 'rmB');
    await request(app).post(`/api/friends/request/${b.user.id}`).set(authHeader(a.token));
    await request(app).post(`/api/friends/accept/${a.user.id}`).set(authHeader(b.token));
    const del = await request(app).delete(`/api/friends/${b.user.id}`).set(authHeader(a.token));
    expect(del.body.removed).toBe(1);
  });
});
