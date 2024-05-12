FROM node:latest
WORKDIR /usr/src/app

COPY package.json package.json
COPY index.js index.js
COPY .npmrc .npmrc

CMD [ "npm", "publish", "--access", "public" ]
