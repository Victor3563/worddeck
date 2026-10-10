# syntax=docker/dockerfile:1

FROM node:22-alpine AS build
WORKDIR /app

COPY package.json package-lock.json ./
ARG NPM_REGISTRY=https://registry.npmjs.org/
RUN npm ci --no-audit --no-fund --registry="$NPM_REGISTRY"

COPY . .
RUN npm run build

FROM nginx:1.28-alpine

# Шаблон рендерится через envsubst при каждом старте контейнера.
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html

ENV NGINX_PORT=80 \
    BACKEND_UPSTREAM=backend:8080 \
    NGINX_RESOLVER=127.0.0.11 \
    NGINX_REAL_IP_FROM=127.0.0.1/32 \
    NGINX_CLIENT_MAX_BODY_SIZE=2m \
    NGINX_AUTH_RATE=10r/m \
    NGINX_AUTH_BURST=20 \
    NGINX_ENVSUBST_FILTER="^(NGINX_|BACKEND_)"

EXPOSE 80
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD wget -qO- "http://127.0.0.1:${NGINX_PORT}/nginx-health" >/dev/null || exit 1
