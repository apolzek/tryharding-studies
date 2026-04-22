FROM node:20-alpine

RUN apk add --no-cache python3 make g++

WORKDIR /app

COPY package.json ./
RUN npm install --omit=dev

COPY src ./src

ENV NODE_ENV=production
ENV PORT=4000
ENV DB_PATH=/data/orkut.db
ENV JWT_SECRET=change-me-in-prod

RUN mkdir -p /data
VOLUME ["/data"]

EXPOSE 4000

CMD ["node", "src/index.js"]
