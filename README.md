# drone-tf-outputs-to-env

[![Build Status](https://travis-ci.org/robertstettner/drone-tf-outputs-to-env.svg?branch=master)](https://travis-ci.org/robertstettner/drone-tf-outputs-to-env)

Drone plugin to execute Terraform output and write to an envfile. For the usage information and
a listing of the available options please take a look at [the docs](https://github.com/robertstettner/drone-tf-outputs-to-env/blob/master/DOCS.md).

## Build

Build the binary with the following commands:

```
export GO111MODULE=on
go mod download
go test
go build
```

## Docker

Build the docker image with the following commands:

```
docker build --rm=true \
  -t robertstettner/drone-tf-outputs-to-env \
  --build-arg terraform_version=0.12.0 .
```

## Usage

Execute from the working directory:

```
docker run --rm \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  robertstettner/drone-tf-outputs-to-env:latest --plan
```
