# Ethereum Node Specs
It is possible to sync and run an Ethereum node in several different ways based on the sync strategy being used, and 
the pruning configuration. This document aims to explain those differences, and what is required for use with 
VulcanizeDB.

_The following descriptions are paraphrased from the [Ethereum](https://ethereum.org/en/developers/docs/nodes-and-clients)
and [Geth](https://geth.ethereum.org/docs/faq) docs._
## Types of Nodes:
- Light: Stores only block headers - all other information needs to be requested from the network.
- Full: Stores all blockchain data including headers, transactions, and receipts. A full node stores current state by
default, and prunes historical state.
- Archive: Stores all blockchain data and an archive of historical states. Historical state is necessary for querying for
state data at a given block in the past, for example, querying for an account balance at block 4,000,000.

## Sync Modes:
- Light: Downloads all block data (including headers) and verifies some randomly.
- Fast (default): Downloads all blocks (including headers, transactions and receipts). It downloads the headers and
the state in separate processes, and verifies both data sets. This sync mode does not process all the blocks, instead
 it only verifies the associated proof-of-works.
        - what this means for VDB is that though the state trie is downloaded, the transactions are not applied, so we aren't getting storage diffs, which we need
- Full: Downloads all blocks (including headers, transactions and receipts) and generates the state of the blockchain
 incrementally by executing every block.

## VulcanizeDB Requirements:
VulcanizeDB currently depends on a [custom Geth implementation](https://github.com/makerdao/go-ethereum/tree/allow-state-diff-subscription) to access it's full feature set.
This custom implementation was developed because we found that fetching contract storage data via the JSON RPC `eth_getStorageAt` method was too slow when syncing data for a large set of contracts such as the Multi-Collateral Dai system contracts. When using `eth_getStorageAt` to sync storage data, an RPC call would need to be made for every storage slot on every contract of interest, for every block. As an alternative to this costly operation, we opted to gather storage diffs (the change in contract storage from block to block) while syncing a custom geth node.

The key change that this implementation makes is a way to subscribe to storage diffs. As a block's transactions are applied, the storage changes are identified and emitted to subscribers. It is also possible to specify contracts to watch when establishing a subscription, so that only diffs from those given contracts will be sent to subscribers. These storage diffs can be used to build up past and current state of domain objects across smart contracts.

### VulcanizeDB Command Requirements:

Several VulcanizeDB top-level commands have specific requirements of the Ethereum node that they connect to, which are detailed below.
- `headerSync`, `execute` and `backfillEvents` are able to connect to any Ethereum node, provided it is not rate limited.
- `extractDiffs` requires the custom Geth patch. The following settings are required when running that node:
    - `gcmode`: The garbage collection mode determines if the Ethereum node prunes data as it is syncing with the chain. Either a `full` or `archive` node is acceptable for VulcanizeDB, though it is strongly recommended to use a `full` node, as the size of the data is much less than an `archive` node. When starting the Geth node, pass the `--gcmode <"full" or "archive">` flag - "full" is the default value.
    - `syncmode`: The `full` sync strategy is required since the custom Geth client streams storage diffs as the node is executing each block. Other syncing strategies do not replay individual transactions as they're syncing, and therefore do not emit storage diffs. The default value is "fast", so it is important to pass the `--syncmode "full"` flag.
- `backfillStorage` requires an Archive node. Backfilling storage values uses the `eth_getStorageAt` JSON RPC call, which needs access to archived data. The node will need to have been synced from the beginning with `--gcmode "archive"`.

 ## Using VulcanizeDB in Light Mode:
 As mentioned above, the full feature set depends on storage diffs to access current and historical state snapshots of Maker domain objects (Ilks, Urns, etc). If you are only interested in accessing raw events logs, it is possible to run VDB against a lighter weight node. When syncing VulcanizeDB without storage diffs enabled, you can remove the storage transformer
 exporters from the config file. Nodes that are considered "lighter weight" include:
 - A node using fast sync mode.
 - A non-patched node.
 - A node using the light syncing strategy. While it is technically possible to sync VDB (both headers and transformed
  log events) with a node running in light mode, this is not recommended. Anecdotally we've seen running VDB against a light
  Ethereum node may be less reliable because finding suitable peers that are configured to serve light nodes was not
  consistent and resulted in getting network errors from RPC calls. For more information, see the following:
    - [https://ethereum.stackexchange.com/questions/11014/how-to-run-a-server-for-light-clients](https://ethereum.stackexchange.com/questions/11014/how-to-run-a-server-for-light-clients)
    - [https://github.com/ethereum/go-ethereum/issues/15454](https://github.com/ethereum/go-ethereum/issues/15454)

