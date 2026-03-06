FROM node:22-alpine AS builder

WORKDIR /build
COPY package.json package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi
COPY . .
RUN npm run build

FROM caddy:2-alpine

COPY --from=builder /build/dist /srv
COPY Caddyfile /etc/caddy/Caddyfile

EXPOSE 3000
