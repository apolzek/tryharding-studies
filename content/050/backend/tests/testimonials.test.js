const { makeApp, registerUser, authHeader, request } = require('./helpers');

describe('testimonials', () => {
  test('friend posts a testimonial, profile owner reads it', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'tsA');
    const b = await registerUser(app, 'tsB');
    const res = await request(app)
      .post(`/api/testimonials/${b.user.id}`)
      .set(authHeader(a.token))
      .send({ body: 'pessoa incrivel' });
    expect(res.status).toBe(201);

    const list = await request(app).get(`/api/testimonials/${b.user.id}`).set(authHeader(b.token));
    expect(list.body).toHaveLength(1);
    expect(list.body[0].author_username).toBe('tsA');
  });

  test('cannot testify for yourself', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'selfT');
    const res = await request(app)
      .post(`/api/testimonials/${a.user.id}`)
      .set(authHeader(a.token))
      .send({ body: 'eu sou top' });
    expect(res.status).toBe(400);
  });

  test('profile owner can delete any testimonial on their wall', async () => {
    const { app } = makeApp();
    const a = await registerUser(app, 'delA');
    const b = await registerUser(app, 'delB');
    const t = await request(app)
      .post(`/api/testimonials/${b.user.id}`)
      .set(authHeader(a.token))
      .send({ body: 'hey' });
    const del = await request(app).delete(`/api/testimonials/${t.body.id}`).set(authHeader(b.token));
    expect(del.status).toBe(200);
  });
});
