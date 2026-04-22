# 050 — Orkut Clone

A faithful-ish clone of classic Orkut. Frontend and backend run in containers.
Auth is username + password only (no email).

## Stack

- **Backend**: Node.js 20, Express, SQLite (better-sqlite3), JWT, bcrypt. Tests with Jest + supertest.
- **Frontend**: React 18 + Vite + React Router. Tests with Vitest + Testing Library.
- **Orchestration**: Docker Compose.

## Features

- Register / login (username + password)
- Editable profile (display name, photo URL, bio, status, age, city, country)
- Scrapbook (post/read/delete scraps on any profile)
- Friend requests, accept, remove, pending list, friend listing
- Communities: create, search, join, leave, list members
- Testimonials (depoimentos) on profiles
- Ratings: trust (confiavel), cool (legal), sexy — 0..3 scale, plus fan button
- Photo album (URL-based)
- Recent visitors tracking

## Quickstart

```bash
docker compose up --build
```

- Frontend: http://localhost:5173
- Backend:  http://localhost:4000 (health: `/health`)

Create a couple of accounts via the register page, then play with scraps,
friends, communities, etc.

## Running tests locally

```bash
# backend
cd backend && npm install && npm test

# frontend
cd frontend && npm install && npm test
```

## Project layout

```
050/
├── backend/
│   ├── src/
│   │   ├── index.js, app.js, db.js, auth.js
│   │   ├── middleware/auth.js
│   │   └── routes/ (auth, users, scraps, friends, communities,
│   │                testimonials, ratings, photos, visits)
│   └── tests/  (Jest + supertest — 32 tests)
├── frontend/
│   ├── src/
│   │   ├── App.jsx, main.jsx, api.js, auth.jsx, styles.css
│   │   ├── components/ (Header, ProfileCard, RatingEditor)
│   │   ├── pages/      (Login, Register, Home, Profile, EditProfile,
│   │   │                Scrapbook, Testimonials, Friends, Communities,
│   │   │                CommunityPage, Photos)
│   │   └── test/       (Vitest — 12 tests)
│   ├── index.html
│   └── vite.config.js
└── docker-compose.yml
```
