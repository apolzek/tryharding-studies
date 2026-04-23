FROM node:20-alpine
WORKDIR /app
COPY services/cart/package.json ./
RUN npm install --omit=dev
COPY services/cart/ ./
EXPOSE 8009
CMD ["node", "-r", "./otel.cjs", "src/server.js"]
