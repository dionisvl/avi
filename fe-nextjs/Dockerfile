# Production-style image: build the Next.js app, run it with `next start`
# (standalone output). Used in the dev stack too — no hot reload, but stable.

# 1. Install dependencies
FROM node:26-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci

# 2. Build. NEXT_PUBLIC_* vars are inlined into the client bundle at build time,
#    so the API URL must be provided here as a build arg.
FROM node:26-alpine AS builder
WORKDIR /app
ARG NEXT_PUBLIC_API_URL=http://api.avi.test
ARG NEXT_PUBLIC_APP_URL=http://avi.test
ARG NEXT_PUBLIC_S3_PUBLIC_BASE_URL=
ARG NEXT_PUBLIC_IMAGE_HOSTS=
ARG NEXT_PUBLIC_DEMO_MODE=false
ARG NEXT_PUBLIC_DEMO_USERS=[]
ARG NEXT_PUBLIC_MAILPIT_URL=
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_APP_URL=$NEXT_PUBLIC_APP_URL
ENV NEXT_PUBLIC_S3_PUBLIC_BASE_URL=$NEXT_PUBLIC_S3_PUBLIC_BASE_URL
ENV NEXT_PUBLIC_IMAGE_HOSTS=$NEXT_PUBLIC_IMAGE_HOSTS
ENV NEXT_PUBLIC_DEMO_MODE=$NEXT_PUBLIC_DEMO_MODE
ENV NEXT_PUBLIC_DEMO_USERS=$NEXT_PUBLIC_DEMO_USERS
ENV NEXT_PUBLIC_MAILPIT_URL=$NEXT_PUBLIC_MAILPIT_URL
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

# 3. Minimal runtime from the standalone output.
FROM node:26-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
EXPOSE 3000
CMD ["node", "server.js"]
