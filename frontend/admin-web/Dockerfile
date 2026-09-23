FROM --platform=$BUILDPLATFORM node:22-alpine AS builder

WORKDIR /app

# Copy root workspace configurations and package files
COPY frontend/package.json ./frontend/
COPY frontend/shared/package.json ./frontend/shared/
COPY frontend/admin-web/package.json ./frontend/admin-web/

WORKDIR /app/frontend
RUN npm install

# Copy source files
WORKDIR /app
COPY frontend/shared ./frontend/shared
COPY frontend/admin-web ./frontend/admin-web

WORKDIR /app/frontend/shared
RUN npm run build

WORKDIR /app/frontend/admin-web
RUN npm run build

# Runtime stage
FROM nginx:1.27-alpine

COPY frontend/admin-web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/frontend/admin-web/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
