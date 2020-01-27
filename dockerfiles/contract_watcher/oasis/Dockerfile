FROM golang:alpine as builder

RUN apk --update --no-cache add g++ git linux-headers

ENV GO111MODULE on

WORKDIR /vulcanizedb
COPY . .
RUN go build


FROM golang:alpine

WORKDIR /app

# add certificates for node requests via https
# bash for wait-for-it.sh
RUN apk update \
        && apk upgrade \
        && apk add --no-cache \
        ca-certificates \
	    bash \
        && update-ca-certificates 2>/dev/null || true

# setup environment
ENV GO111MODULE on

# Direct logs to stdout for docker log driver
RUN ln -sf /dev/stdout /app/vulcanizedb.log

# add required files
COPY --from=builder /vulcanizedb/vulcanizedb .
COPY --from=builder /vulcanizedb/environments/oasisContractWatcher.toml .
# needed for waiting until postgres is ready before starting from docker-compose
COPY --from=builder /vulcanizedb/dockerfiles/wait-for-it.sh .

# need to execute with a shell to access env variables
CMD ["./vulcanizedb", "contractWatcher", "--config", "oasisContractWatcher.toml"]