# Build stage
FROM node:22-alpine AS build
WORKDIR /app

# Install dependencies via npm (pixi handles the outer layer)
COPY package*.json ./
RUN npm install

# Copy source code and build
COPY . .
RUN npm run build

# Production serve stage using NGINX
FROM nginx:alpine
# Remove default nginx static assets
RUN rm -rf /usr/share/nginx/html/*

# Copy built assets from builder
COPY --from=build /app/dist /usr/share/nginx/html

# Copy custom NGINX configuration for SPA routing
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
