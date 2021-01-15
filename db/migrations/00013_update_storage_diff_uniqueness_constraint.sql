-- +goose Up
ALTER TABLE public.storage_diff
   DROP CONSTRAINT storage_diff_block_height_block_hash_address_storage_key_st_key,
   ADD UNIQUE (block_height, block_hash, address, storage_key, storage_value, status);

-- +goose Down
ALTER TABLE public.storage_diff
    DROP CONSTRAINT storage_diff_block_height_block_hash_address_storage_key_st_key,
    ADD UNIQUE (block_height, block_hash, address, storage_key, storage_value);
