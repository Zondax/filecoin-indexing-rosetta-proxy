module github.com/zondax/filecoin-indexing-rosetta-proxy

go 1.16

require (
	github.com/Zondax/zindexer v0.1.2
	github.com/coinbase/rosetta-sdk-go v0.6.10
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-state-types v0.1.1-0.20210506134452-99b279731c48
	github.com/filecoin-project/lotus v1.10.0
	github.com/filecoin-project/specs-actors/v4 v4.0.0
	github.com/filecoin-project/specs-actors/v5 v5.0.1
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-log v1.0.4
	github.com/orcaman/concurrent-map v0.0.0-20190826125027-8c72a8bb44f6
	github.com/spf13/viper v1.7.1
	github.com/zondax/rosetta-filecoin-lib v1.1000.1
	github.com/zondax/rosetta-filecoin-proxy v1.1000.1
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
