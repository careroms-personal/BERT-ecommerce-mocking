FROM node:22-alpine AS builder

WORKDIR /app

# Install all deps (including devDeps for tsc)
COPY package.json tsconfig.json ./
RUN npm install

COPY src ./src
RUN npm run build

# ─────────────────────────────────────────────
FROM node:22-alpine

RUN apk add --no-cache curl tzdata

WORKDIR /app

# Install production deps only
COPY package.json ./
RUN npm install --omit=dev

# Copy compiled output
COPY --from=builder /app/dist ./dist

ENV NODE_ENV=production
ENV PRETTY_LOGS=false

EXPOSE 8083

CMD ["node", "dist/index.js"]
