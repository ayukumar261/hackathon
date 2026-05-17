FROM node:20-alpine

RUN corepack enable && corepack prepare pnpm@9.0.0 --activate

WORKDIR /app

COPY package.json ./
RUN pnpm install

EXPOSE 3000

CMD ["pnpm", "dev"]
