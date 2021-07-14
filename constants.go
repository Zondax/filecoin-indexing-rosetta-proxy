package main

var (
	// Versions info to be injected on build time
	RosettaSDKVersion = "Unknown"
	LotusVersion      = "Unknown"
	GitRevision       = "Unknown"

	// RosettaServerPort ServerPort to be injected on build time
	RosettaServerPort = "8080"

	// Other configs
	RetryConnectAttempts = "1000000"
)

const (
	// Network
	BlockChainName = "Filecoin"
	NetworkName    = "mainnet"
)
