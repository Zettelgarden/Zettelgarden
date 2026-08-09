# Dockerfile-frontend
# Build context is the REPO ROOT (see docker-compose.yml) so the frontend
# build can reach the shared sync engine source (sync-engine/) that the
# data-provider seam imports directly (Zettelgarden-v5b Phase 2b).
FROM node:18

WORKDIR /usr/src/app

# Install frontend dependencies first (cache-friendly).
COPY zettelkasten-front/package*.json ./zettelkasten-front/
RUN cd zettelkasten-front && npm install

# Copy the frontend source and the sync engine it imports by path.
COPY zettelkasten-front/ ./zettelkasten-front/
COPY sync-engine/src/ ./sync-engine/src/
COPY sync-engine/package.json ./sync-engine/

# Vite inlines every VITE_* var at BUILD time (import.meta.env.* is statically
# replaced), so each one must be declared as an ARG here AND passed on the
# build command. Defaults keep OIDC opt-in (off) unless compose overrides it.
ARG VITE_URL
ARG VITE_ENV
ARG VITE_OIDC_ENABLED=false
ARG VITE_OIDC_LABEL
ARG VITE_GITHUB_AUTH_ENABLED=true
RUN cd zettelkasten-front && \
    VITE_URL=${VITE_URL} VITE_ENV=${VITE_ENV} \
    VITE_OIDC_ENABLED=${VITE_OIDC_ENABLED} \
    VITE_OIDC_LABEL=${VITE_OIDC_LABEL} \
    VITE_GITHUB_AUTH_ENABLED=${VITE_GITHUB_AUTH_ENABLED} \
    npm run build

RUN npm install -g serve

EXPOSE 3000

# Serve the built SPA from the repo-root workdir.
CMD ["serve", "-s", "/usr/src/app/zettelkasten-front/dist", "-l", "3000"]
