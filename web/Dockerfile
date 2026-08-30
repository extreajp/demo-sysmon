FROM node:22-alpine AS deps
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm ci

FROM node:22-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY web/ .
ARG NEXT_PUBLIC_SYSMON_URL=http://127.0.0.1:9101
ENV NEXT_PUBLIC_SYSMON_URL=$NEXT_PUBLIC_SYSMON_URL
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV HOME=/tmp
ENV CPU_WORKER_PATH=/app/cpu-worker.js
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/cpu-worker.js ./cpu-worker.js
USER nobody
EXPOSE 3000
CMD ["node", "server.js"]
