module github.com/zondax/filecoin-indexing-rosetta-proxy

go 1.16

require (
	github.com/coinbase/rosetta-sdk-go v0.6.10
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-state-types v0.1.1-0.20210506134452-99b279731c48
	github.com/filecoin-project/lotus v1.10.0
	github.com/filecoin-project/specs-actors/v4 v4.0.0
	github.com/ipfs/go-log v1.0.4
	github.com/zondax/rosetta-filecoin-lib v1.1000.1
	github.com/zondax/rosetta-filecoin-proxy v1.1000.1
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
