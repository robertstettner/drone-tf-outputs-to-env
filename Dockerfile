# Docker image for the Drone Terraform Outputs to Env plugin
#
#     docker build -t robertstettner/drone-tf-outputs-to-env:latest .
FROM golang:1.13-alpine AS builder

RUN apk add --no-cache git

RUN mkdir -p /tmp/drone-tf-outputs-to-env
WORKDIR /tmp/drone-tf-outputs-to-env

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -o /go/bin/drone-tf-outputs-to-env

FROM alpine:3.9

RUN apk -U add \
  ca-certificates \
  git \
  wget \
  openssh-client && \
  rm -rf /var/cache/apk/*

ARG terraform_version
RUN wget -q https://releases.hashicorp.com/terraform/${terraform_version}/terraform_${terraform_version}_linux_amd64.zip -O terraform.zip && \
  unzip terraform.zip -d /bin && \
  rm -f terraform.zip

COPY --from=builder /go/bin/drone-tf-outputs-to-env /bin/
ENTRYPOINT ["/bin/drone-tf-outputs-to-env"]
