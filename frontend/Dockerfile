FROM node:24-alpine AS builder
WORKDIR /app

ARG VITE_API_BASE_URL=/api
ARG VITE_KEYCLOAK_URL=http://localhost/account
ARG VITE_KEYCLOAK_REALM=quiz-rush
ARG VITE_KEYCLOAK_CLIENT_ID=quiz-rush-app

ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_KEYCLOAK_URL=$VITE_KEYCLOAK_URL
ENV VITE_KEYCLOAK_REALM=$VITE_KEYCLOAK_REALM
ENV VITE_KEYCLOAK_CLIENT_ID=$VITE_KEYCLOAK_CLIENT_ID

COPY package.json pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile

COPY . .
RUN pnpm build

FROM nginx:1.29-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
