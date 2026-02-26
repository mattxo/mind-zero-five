FROM golang:1.24-alpine AS builder
RUN apk add --no-cache ca-certificates git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server
RUN CGO_ENABLED=0 go build -o mind ./cmd/mind
RUN CGO_ENABLED=0 go build -o eg ./cmd/eg
RUN GOOS=js GOARCH=wasm go build -o web/ui.wasm ./cmd/ui
RUN cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js

FROM golang:1.24-alpine
RUN apk add --no-cache ca-certificates nodejs npm git openssh-client bash tmux su-exec
RUN npm install -g @anthropic-ai/claude-code

RUN adduser -D -h /home/app app

COPY --from=builder /app/server /usr/local/bin/server
COPY --from=builder /app/mind /usr/local/bin/mind
COPY --from=builder /app/eg /usr/local/bin/eg
COPY --from=builder /app/web /home/app/web
COPY --from=builder /app /usr/local/share/mz5-source
RUN cd /usr/local/share/mz5-source && go mod download

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN sed -i 's/\r$//' /usr/local/bin/entrypoint.sh && chmod +x /usr/local/bin/entrypoint.sh

RUN mkdir -p /data/repos && chown -R app:app /data
RUN chown -R app:app /home/app /usr/local/share/mz5-source

WORKDIR /home/app
ENV WASM_DIR=/home/app/web
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["server"]
