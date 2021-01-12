-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY event_logs_transformed
    ON event_logs (transformed)
    WHERE transformed = true;

-- +goose Down
DROP INDEX event_logs_transformed;
