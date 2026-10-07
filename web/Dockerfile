FROM oven/bun:1.4.2
WORKDIR /app
COPY package.json bun.lock ./
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
RUN bun install --frozen-lockfile
COPY . .
RUN bun run build
EXPOSE 5173
CMD ["bun", "src/server.ts"]
