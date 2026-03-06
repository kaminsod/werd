FROM node:22-alpine AS builder

WORKDIR /build
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
RUN npm run build

FROM caddy:2-alpine

COPY --from=builder /build/dist /srv
COPY <<EOF /etc/caddy/Caddyfile
:3000 {
    root * /srv
    file_server
    try_files {path} /index.html
}
EOF

EXPOSE 3000
