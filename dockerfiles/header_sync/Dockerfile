FROM makerdao/vdb-builder:latest as builder

WORKDIR /vulcanizedb
COPY . .
RUN go build

FROM makerdao/vdb-runner:latest as runner

WORKDIR /app

# Direct logs to stdout for docker log driver
RUN ln -sf /dev/stdout /app/vulcanizedb.log

# add required files
COPY --from=builder /vulcanizedb/vulcanizedb .
COPY --from=builder /vulcanizedb/Makefile .
COPY --from=builder /vulcanizedb/dockerfiles/header_sync/startup_script.sh .
COPY --from=builder /vulcanizedb/db/migrations/* db/migrations/
COPY --from=builder /go/bin/goose goose
# needed for waiting until postgres is ready before starting from docker-compose
COPY --from=builder /vulcanizedb/dockerfiles/wait-for-it.sh .

# NOTE: You may need to update this in your cloud provider config in addition to changes here
HEALTHCHECK CMD test -f /tmp/header_sync_health_check

# need to execute with a shell to access env variables
CMD ["./startup_script.sh"]
