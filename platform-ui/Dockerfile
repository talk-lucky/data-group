# Stage 1: Build the Vue.js application
FROM node:18-alpine AS builder

# Set working directory
WORKDIR /app

# Copy package.json and package-lock.json (or npm-shrinkwrap.json if available)
# If using yarn, copy yarn.lock
COPY package.json ./
# Assuming npm, if you have a package-lock.json, copy it too for reproducible builds
# COPY package-lock.json ./

# Install dependencies
# Using --legacy-peer-deps from previous successful installs
RUN npm install --legacy-peer-deps

# Copy the rest of the application files
COPY . .

# Set VUE_APP_API_BASE_URL at build time if needed, or rely on .env files copied
# For this setup, we assume .env.development (or .env.production if building for prod)
# is correctly configured and will be used by `npm run build`.
# If you want to override .env files or provide a default if they are missing:
# ARG VUE_APP_API_BASE_URL=/api/v1 # Default if not provided
# ENV VUE_APP_API_BASE_URL=${VUE_APP_API_BASE_URL}

# Build the application for production
RUN npm run build

# Stage 2: Serve the built application with Nginx
FROM nginx:stable-alpine

# Set working directory for Nginx
WORKDIR /usr/share/nginx/html

# Remove default Nginx static assets
RUN rm -rf ./*

# Copy built assets from the builder stage
COPY --from=builder /app/dist ./

# Copy custom Nginx configuration (will be created in the next step)
# This nginx.conf should be in the same directory as this Dockerfile (platform-ui/)
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port 80 (Nginx default HTTP port)
EXPOSE 80

# Start Nginx
CMD ["nginx", "-g", "daemon off;"]
