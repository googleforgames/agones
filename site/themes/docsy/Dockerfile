FROM klakegg/hugo:0.101.0-ext-alpine as docsy-user-guide

RUN apk update
RUN apk add git
COPY package.json /app/docsy/userguide/
WORKDIR /app/docsy/userguide/
RUN npm install --production=false
RUN git config --global --add safe.directory /app/docsy

CMD ["serve", "--cleanDestinationDir", "--themesDir", "../..", "--baseURL",  "http://localhost:1313/", "--buildDrafts", "--buildFuture", "--disableFastRender", "--ignoreCache", "--watch"]
