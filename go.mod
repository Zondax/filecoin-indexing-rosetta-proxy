module github.com/zondax/filecoin-indexing-rosetta-proxy

go 1.16

require (
	github.com/Zondax/zindexer v0.2.11
	github.com/coinbase/rosetta-sdk-go v0.7.2
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/go-jsonrpc v0.1.5
	github.com/filecoin-project/go-state-types v0.1.3
	github.com/filecoin-project/lotus v1.15.1
	github.com/filecoin-project/specs-actors/v7 v7.0.0
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-log v1.0.5
	github.com/orcaman/concurrent-map v1.0.0
	github.com/spf13/viper v1.7.1
	github.com/zondax/rosetta-filecoin-lib v1.1501.0
	github.com/zondax/rosetta-filecoin-proxy v1.1501.0
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
