# syntax=docker/dockerfile:1

FROM oven/bun:1 AS build
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile
COPY . .
ARG VITE_API_URL=http://localhost:8080/api/v1
ENV VITE_API_URL=$VITE_API_URL
RUN bun run build

FROM oven/bun:1
WORKDIR /app
COPY --from=build /app /app
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["bun", "run", "preview", "--", "--host", "0.0.0.0", "--port", "3000"]
