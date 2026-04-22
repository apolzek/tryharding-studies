const { makeApp, registerUser, authHeader, request } = require('./helpers');

describe('photos', () => {
  test('upload, list, delete photo', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'phA');

    const up = await request(app)
      .post('/api/photos')
      .set(authHeader(a.token))
      .send({ url: 'https://cdn.example/1.jpg', caption: 'festa' });
    expect(up.status).toBe(201);

    const list = await request(app).get(`/api/photos/${a.user.id}`).set(authHeader(a.token));
    expect(list.body).toHaveLength(1);

    const del = await request(app).delete(`/api/photos/${up.body.id}`).set(authHeader(a.token));
    expect(del.status).toBe(200);
  });

  test('cannot delete another user photo', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'phB');
    const b = await registerUser(app, 'phC');
    const up = await request(app).post('/api/photos').set(authHeader(a.token)).send({ url: 'u' });
    const del = await request(app).delete(`/api/photos/${up.body.id}`).set(authHeader(b.token));
    expect(del.status).toBe(403);
  });
});
