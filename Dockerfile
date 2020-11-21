FROM golang:latest as builder

ARG KUBECTL_VERSION="1.18.8"
ARG KUSTOMIZE_VERSION="3.8.1"

WORKDIR /app

COPY . .

RUN mkdir -p /app/bin \
    && curl -Lo /app/bin/kubectl "https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl" \
    && chmod +x /app/bin/kubectl \
    && curl -L "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv${KUSTOMIZE_VERSION}/kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz" | tar xzvf - -C /app/bin \
    && chmod +x /app/bin/kustomize

ARG GOOS=linux
ARG GOARCH=amd64

RUN cd cmd/kubectl-clean \
    && CGO_ENABLED=0 go build -mod vendor -ldflags "-extldflags '-static'"  -o /app/bin/kubectl-clean .

FROM alpine

RUN apk add --no-cache --update ca-certificates \
    && rm -rf /tmp/* \
    && rm -rf /var/cache/apk/* \
    && rm -rf /var/tmp/*

COPY --from=builder /app/bin/ /usr/local/bin/

ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [""]