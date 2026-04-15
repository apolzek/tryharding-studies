FROM node:20-alpine AS build
WORKDIR /app
ARG VITE_OTLP_ENDPOINT=http://localhost:24318
ARG VITE_FARO_URL=http://localhost:12347/collect
ARG VITE_REPLAY_ENDPOINT=http://localhost:18080/replay/ingest
ENV VITE_OTLP_ENDPOINT=$VITE_OTLP_ENDPOINT
ENV VITE_FARO_URL=$VITE_FARO_URL
ENV VITE_REPLAY_ENDPOINT=$VITE_REPLAY_ENDPOINT
COPY package.json ./
RUN npm install --no-audit --no-fund
COPY . .
RUN node build.js

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
