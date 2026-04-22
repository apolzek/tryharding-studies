# 053 — SRE Challenge Platform

Plataforma tipo CKA (Certified Kubernetes Administrator) para desafios práticos de SRE.
Cada desafio roda em um container Docker isolado, acessível pelo browser via terminal web.

## Stack & projetos open-source usados

| Peça | Projeto OSS | Papel |
|------|-------------|------|
| Terminal no browser | [**ttyd**](https://github.com/tsl0922/ttyd) | Expõe `bash` do container via WebSocket/HTTP |
| Orquestração dinâmica | [**dockerode**](https://github.com/apocas/dockerode) | Backend cria/remove containers de desafio |
| UI | React + Vite | Portal de desafios + admin |
| Auth admin | `jsonwebtoken` + `bcryptjs` | Login do admin |
| Persistência | `better-sqlite3` | Desafios, sessões, usuários |

Outros projetos OSS equivalentes a `ttyd` que também funcionam nesta arquitetura:
[wetty](https://github.com/butlerx/wetty), [gotty](https://github.com/sorenisanerd/gotty), [xterm.js](https://xtermjs.org/).

## Arquitetura

```
 Browser ──► Frontend (React, :8053)
              │
              ▼
           Backend (Node/Express, :8054) ──► Docker Engine (unix:///var/run/docker.sock)
                                               │
                                               ▼
                                    Challenge Container N
                                    (ubuntu + ttyd + setup.sh)
                                    exposto em :9100..9199
```

Quando o usuário inicia um desafio:
1. Backend aloca uma porta livre em `9100-9199`.
2. Cria container a partir da imagem `sre-challenge-base` com `setup.sh` e `verify.sh` injetados.
3. `entrypoint.sh` roda o setup (quebra o sistema de propósito) e sobe `ttyd bash` na porta.
4. Frontend abre um iframe em `http://<host>:<porta>` com o terminal.
5. Botão **Verify** chama `docker exec` com `verify.sh`; exit code 0 = passou.

## Portas (evita conflito com 050 que usa 4000/5173)

| Serviço | Porta host |
|---------|-----------|
| Frontend | **8053** |
| Backend | **8054** |
| Desafios | **9100-9199** (dinâmico) |

## Subir

```bash
# Da raiz de content/053
docker compose up -d --build

# Frontend
open http://localhost:8053

# Admin (default):  admin / admin123  — troque em produção
open http://localhost:8053/admin
```

Na primeira subida o backend:
- Constrói a imagem `sre-challenge-base:latest`.
- Popula o SQLite com **30+ desafios** do `backend/seed/challenges.json`.
- Cria o usuário admin se ele ainda não existir.

## Estrutura de um desafio

```json
{
  "slug": "nginx-502",
  "title": "Nginx retornando 502",
  "category": "web",
  "difficulty": "easy",
  "time_limit_sec": 900,
  "objective": "Fazer com que `curl localhost` retorne HTTP 200.",
  "hints": ["cheque o status do nginx", "leia /etc/nginx/nginx.conf"],
  "setup_script": "bash inline que quebra o sistema",
  "verify_script": "bash inline; exit 0 = passou"
}
```

Admin UI permite criar/editar/deletar tudo pela tela `/admin`.

## 30+ desafios inclusos

Divididos em categorias **linux**, **networking**, **web**, **db**, **docker**, **kubernetes**, **observability**.
Ver `backend/seed/challenges.json` ou a lista no portal.

## Limpeza

```bash
docker compose down -v
# Containers órfãos de desafios (caso backend tenha morrido sem limpar):
docker ps --filter "label=sre-challenge=true" -q | xargs -r docker rm -f
```
