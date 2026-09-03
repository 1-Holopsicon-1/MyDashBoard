FROM node:24-alpine AS build
RUN corepack enable
WORKDIR /src
COPY package.json yarn.lock .yarnrc.yml ./
COPY .yarn .yarn
RUN yarn install --immutable
COPY . .
RUN yarn build

FROM caddy:2.8-alpine
COPY --from=build /src/build /srv
COPY Caddyfile /etc/caddy/Caddyfile
RUN mkdir -p /data /config && chown -R caddy:caddy /data /config
EXPOSE 8080
USER caddy
