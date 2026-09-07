FROM node:22-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
ENV PORT=3000
ENV DATABASE_URL=${DATABASE_URL}
EXPOSE 3000
CMD ["npm", "start"]