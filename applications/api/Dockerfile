FROM node:22-alpine
WORKDIR /app

COPY package*.json ./
RUN npm ci --omit=dev

COPY . .

# Run as non-root user for Kubernetes security compliance
RUN chown -R node:node /app
USER node

ENV PORT=3005
EXPOSE 3005

# Run node directly so SIGTERM is handled without npm wrapper
CMD ["node", "src/index.js"]