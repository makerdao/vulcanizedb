-- +goose Up
create index storage_diff_checked_index on storage_diff(checked);


-- +goose Down
drop index storage_diff_checked_index;

