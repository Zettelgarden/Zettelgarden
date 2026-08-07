# Dockerfile-frontend
# Use an official Node runtime as a parent image
FROM node:18

# Set the working directory in the container
WORKDIR /usr/src/app

# Copy package.json and package-lock.json
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy the current directory contents into the container at /usr/src/app
COPY . .

# Set environment variables and run the build.
# Vite inlines every VITE_* var at BUILD time (import.meta.env.* is statically
# replaced), so each one must be declared as an ARG here AND passed on the
# build command — a runtime `environment:` block can't reach already-built
# static assets. Defaults keep OIDC opt-in (off) unless compose overrides it.
ARG VITE_URL
ARG VITE_ENV
ARG VITE_OIDC_ENABLED=false
ARG VITE_OIDC_LABEL
ARG VITE_GITHUB_AUTH_ENABLED=true
RUN VITE_URL=${VITE_URL} VITE_ENV=${VITE_ENV} \
    VITE_OIDC_ENABLED=${VITE_OIDC_ENABLED} \
    VITE_OIDC_LABEL=${VITE_OIDC_LABEL} \
    VITE_GITHUB_AUTH_ENABLED=${VITE_GITHUB_AUTH_ENABLED} \
    npm run build

# Use serve to serve the build folder
RUN npm install -g serve

# Make port 3000 available to the world outside this container
EXPOSE 3000

# Run serve when the container launches
CMD ["serve", "-s", "dist", "-l", "3000"]
