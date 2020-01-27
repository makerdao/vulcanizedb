# Docker

## headerSync
Sync block headers.

#### Build
From project root directory:
```
docker build -f dockerfiles/header_sync/Dockerfile . -t header_sync:latest
```

#### Run
```
docker run -e DATABASE_USER=user -e DATABASE_PASSWORD=password -e DATABASE_HOSTNAME=host -e DATABASE_PORT=port -e DATABASE_NAME=name -e STARTING_BLOCK_NUMBER=0 -e CLIENT_IPCPATH=path -it header_sync:latest
```
Note: must replace env var values with appropriate replacements given your database/node setup.

## contractWatcher
Sync events from given contract(s).

Note: depends on separate headerSync process persisting headers to same database.

### Oasis

#### Build
From project root directory:
```
docker build -f dockerfiles/contract_watcher/oasis/Dockerfile . -t contract_watcher:oasis-latest
```

#### Run
```
docker run -e DATABASE_USER=user -e DATABASE_PASSWORD=password -e DATABASE_HOSTNAME=host -e DATABASE_PORT=port -e DATABASE_NAME=name -e CLIENT_IPCPATH=path -it contract_watcher:oasis-latest
```
Note: must replace env var values with appropriate replacements given your database/node setup.
