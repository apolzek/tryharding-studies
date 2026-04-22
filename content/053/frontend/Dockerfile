FROM node:20-alpine

WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install

COPY index.html vite.config.js ./
COPY src ./src

EXPOSE 8053
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0", "--port", "8053"]
