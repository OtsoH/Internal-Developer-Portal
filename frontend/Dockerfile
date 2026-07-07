# syntax=docker/dockerfile:1

# --- dev: hot reload; source is bind-mounted by docker-compose, with
# anonymous volumes shadowing /app/node_modules and /app/.next so the
# Linux-built dependencies never mix with the host's ---
FROM node:20-alpine AS dev
WORKDIR /app
RUN npm i -g pnpm@10.34.4
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
CMD ["pnpm", "dev"]
