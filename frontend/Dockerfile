FROM node:24-alpine AS build
RUN corepack enable
WORKDIR /src
COPY package.json yarn.lock .yarnrc.yml ./
RUN yarn install --immutable
COPY . .
RUN yarn build

FROM caddy:2.8-alpine
COPY --from=build /src/build /srv
COPY Caddyfile /etc/caddy/Caddyfile
RUN adduser -D -u 10001 caddy && mkdir -p /data /config && chown -R caddy:caddy /data /config /srv /etc/caddy
EXPOSE 8080
USER caddy
