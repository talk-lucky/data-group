# Use the official Nginx image from Docker Hub
FROM nginx:latest

# Remove the default Nginx configuration file that might conflict
# RUN rm /etc/nginx/conf.d/default.conf

# Copy the custom site configuration file from the host's conf.d directory
# into the Nginx container's configuration directory.
# The source path is relative to the Docker build context (which will be nginx_gateway/)
COPY conf.d/default.conf /etc/nginx/conf.d/default.conf

# Expose the port Nginx will listen on (as defined in default.conf)
EXPOSE 8888

# Command to run Nginx in the foreground
CMD ["nginx", "-g", "daemon off;"]
