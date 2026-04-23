FROM node:20-alpine
WORKDIR /app
COPY services/bff/package.json ./
RUN npm install --omit=dev
COPY services/bff/ ./
COPY proto/ ./proto/
EXPOSE 3000
CMD ["node", "-r", "./otel.cjs", "src/server.js"]
