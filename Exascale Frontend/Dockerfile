# syntax=docker/dockerfile:1.6
# Exascale web — Nuxt 4 SSR + the Nitro BFF (server/api/** proxies to the platform services in-cluster).
# Multi-stage: build the .output bundle, then run it on a minimal node image as non-root.
#
# Upstream service URLs are runtimeConfig (server-only) and are set at RUNTIME via NUXT_* env in the
# k8s ConfigMap (deploy/k8s/web/base/configmap.yaml) — the image itself is environment-agnostic.
FROM node:20-alpine AS build
WORKDIR /app
# Install deps first (cached layer). Fall back to `npm install` when there's no lockfile.
COPY package.json package-lock.json* ./
RUN npm ci --no-audit --no-fund || npm install --no-audit --no-fund
COPY . .
ENV NODE_ENV=production
RUN npm run build

FROM node:20-alpine AS run
WORKDIR /app
ENV NODE_ENV=production \
    NITRO_PORT=3000 \
    NITRO_HOST=0.0.0.0
# Only the built server bundle ships — no source, no node_modules, no dev deps.
COPY --from=build /app/.output ./.output
EXPOSE 3000
USER node
CMD ["node", ".output/server/index.mjs"]
