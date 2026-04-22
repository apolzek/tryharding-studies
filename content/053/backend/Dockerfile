FROM node:20-bookworm-slim

# docker CLI: makes debugging from inside the container easy; dockerode talks
# to the socket directly but `docker ps` is convenient.
RUN apt-get update && apt-get install -y --no-install-recommends \
      docker.io ca-certificates tini \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install --omit=dev

COPY src ./src
COPY seed ./seed

ENV NODE_ENV=production
EXPOSE 8054

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["node", "src/index.js"]
