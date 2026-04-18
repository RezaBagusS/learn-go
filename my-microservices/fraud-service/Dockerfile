FROM node:20-alpine

WORKDIR /app

# Copy package files
COPY fraud-service/package*.json ./

RUN npm install --production

# Copy fraud-service content
COPY fraud-service/ .

# Copy shared folder (untuk proto)
COPY shared/ ../shared/

EXPOSE 50051

CMD ["node", "index.js"]
