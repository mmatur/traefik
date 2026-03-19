FROM node:24-alpine3.23
# Current Active LTS release according to (https://nodejs.org/en/about/releases/)

ENV WEBUI_DIR=/src/webui
RUN mkdir -p $WEBUI_DIR

COPY package.json $WEBUI_DIR/
COPY yarn.lock $WEBUI_DIR/
COPY .yarnrc.yml $WEBUI_DIR/
COPY .yarn/ $WEBUI_DIR/.yarn/

WORKDIR $WEBUI_DIR
RUN corepack enable && yarn install

COPY . $WEBUI_DIR/

EXPOSE 8080

RUN yarn lint
