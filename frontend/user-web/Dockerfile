# Multi-stage Dockerfile for user-web
FROM node:22-alpine AS builder

WORKDIR /app

# Copy root workspace configurations and package files
COPY frontend/package.json ./frontend/
COPY frontend/shared/package.json ./frontend/shared/
COPY frontend/user-web/package.json ./frontend/user-web/

WORKDIR /app/frontend
RUN npm install

# Copy source files
WORKDIR /app
COPY frontend/shared ./frontend/shared
COPY frontend/user-web ./frontend/user-web

WORKDIR /app/frontend/shared
RUN npm run build

WORKDIR /app/frontend/user-web
RUN npm run build

# Runtime stage
FROM nginx:1.27-alpine

COPY frontend/user-web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/frontend/user-web/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
