const { makeApp, registerUser, authHeader, request } = require('./helpers');

describe('scraps', () => {
  test('post and list scraps', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'aaa');
    const b = await registerUser(app, 'bbb');
    const postRes = await request(app)
      .post(`/api/scraps/${b.user.id}`)
      .set(authHeader(a.token))
      .send({ body: 'oi, tudo bem?' });
    expect(postRes.status).toBe(201);

    const listRes = await request(app).get(`/api/scraps/${b.user.id}`).set(authHeader(b.token));
    expect(listRes.status).toBe(200);
    expect(listRes.body).toHaveLength(1);
    expect(listRes.body[0].body).toBe('oi, tudo bem?');
    expect(listRes.body[0].author_username).toBe('aaa');
  });

  test('empty scrap is rejected', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'aa2');
    const b = await registerUser(app, 'bb2');
    const res = await request(app)
      .post(`/api/scraps/${b.user.id}`)
      .set(authHeader(a.token))
      .send({ body: '   ' });
    expect(res.status).toBe(400);
  });

  test('author can delete own scrap', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'aa3');
    const b = await registerUser(app, 'bb3');
    const created = await request(app)
      .post(`/api/scraps/${b.user.id}`)
      .set(authHeader(a.token))
      .send({ body: 'hey' });
    const del = await request(app)
      .delete(`/api/scraps/${created.body.id}`)
      .set(authHeader(a.token));
    expect(del.status).toBe(200);
  });

  test('third party cannot delete scrap', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'aa4');
    const b = await registerUser(app, 'bb4');
    const c = await registerUser(app, 'cc4');
    const created = await request(app)
      .post(`/api/scraps/${b.user.id}`)
      .set(authHeader(a.token))
      .send({ body: 'hey' });
    const del = await request(app)
      .delete(`/api/scraps/${created.body.id}`)
      .set(authHeader(c.token));
    expect(del.status).toBe(403);
  });
});
