FROM node:20-alpine

WORKDIR /app

COPY package.json ./
RUN npm install --no-audit --no-fund

COPY vite.config.js index.html ./
COPY src ./src

EXPOSE 5173

CMD ["npm", "run", "dev"]
