FROM golang:1.26-bookworm AS go-runtime

FROM node:22-bookworm
WORKDIR /app

ENV GOTOOLCHAIN=local \
    GOPROXY=https://goproxy.cn|https://proxy.golang.org|direct \
    npm_config_update_notifier=false \
    PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

RUN apt-get update -o Acquire::Retries=5 \
    && apt-get install -y --no-install-recommends -o Acquire::Retries=5 \
        bash \
        ca-certificates \
        curl \
        git \
        openssh-client \
        python3 \
        python3-pip \
        jq \
        ripgrep \
        sqlite3 \
        unzip \
        zip \
        build-essential \
    && rm -rf /var/lib/apt/lists/* \
    && ln -sf /usr/bin/python3 /usr/local/bin/python \
    && git config --system user.email aiop-orchestrator@example.local \
    && git config --system user.name aiop-orchestrator \
    && git config --system --add safe.directory /workspace/default \
    && printf 'export PATH=/usr/local/go/bin:$PATH\n' > /etc/profile.d/go.sh

RUN npm install -g \
        @openai/codex@0.132.0 \
        @anthropic-ai/claude-code@2.1.150 \
        @google/gemini-cli@0.43.0 \
    && npm cache clean --force

COPY --from=go-runtime /usr/local/go /usr/local/go
COPY --chmod=755 bin/aiop-server-linux-amd64 /app/server
COPY web/dist /app/web/dist
COPY config.container.yaml /app/config.container.yaml

EXPOSE 8080
VOLUME ["/data", "/workspace/default"]

CMD ["/app/server", "--config", "/app/config.container.yaml"]
