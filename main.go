package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Zondax/zindexer/components/connections/data_store"
	filparser "github.com/zondax/fil-parser/parser"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/services"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/services/call"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	rosettaFilecoinLib "github.com/zondax/rosetta-filecoin-lib"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"

	rosettaAsserter "github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	logging "github.com/ipfs/go-log"
	"github.com/spf13/viper"
)

var (
	BlockchainName = tools.BlockChainName
	ServerPort, _  = strconv.Atoi(tools.RosettaServerPort)
)

func logVersionsInfo() {
	rosetta.Logger.Info("********| filecoin-indexing-rosetta-proxy |*********")
	rosetta.Logger.Infof("Rosetta SDK version: %s", tools.RosettaSDKVersion)
	rosetta.Logger.Infof("Lotus version: %s", tools.LotusVersion)
	rosetta.Logger.Infof("Git revision: %s", tools.GitRevision)
	rosetta.Logger.Info("****************************************************")
}

func startLogger(level string) {
	lvl, err := logging.LevelFromString(level)
	if err != nil {
		panic(err)
	}
	logging.SetAllLoggers(lvl)
}

// nolint
func getFullNodeAPI(addr string, token string) (api.FullNode, jsonrpc.ClientCloser, error) {
	headers := http.Header{}
	if len(token) > 0 {
		headers.Add("Authorization", "Bearer "+token)
	}

	return client.NewFullNodeRPCV1(context.Background(), addr, headers)
}

// newBlockchainRouter creates a Mux http.Handler from a collection
// of server controllers.
func newBlockchainRouter(
	network *types.NetworkIdentifier,
	asserter *rosettaAsserter.Asserter,
	api api.FullNode,
	traceRetriever *tools.TraceRetriever,
	rosettaLib *rosettaFilecoinLib.RosettaConstructionFilecoin,
) http.Handler {
	accountAPIService := rosetta.NewAccountAPIService(network, &api, rosettaLib)
	accountAPIController := server.NewAccountAPIController(
		accountAPIService,
		asserter,
	)

	networkAPIService := rosetta.NewNetworkAPIService(network, &api, filparser.GetSupportedOps())
	networkAPIController := server.NewNetworkAPIController(
		networkAPIService,
		asserter,
	)

	blockAPIService := services.NewBlockAPIService(network, &api, traceRetriever, rosettaLib)
	blockAPIController := server.NewBlockAPIController(
		blockAPIService,
		asserter,
	)

	callAPIService := call.NewCallAPIService(network, &api, traceRetriever)
	callAPIController := server.NewCallAPIController(
		callAPIService,
		asserter,
	)

	mempoolAPIService := rosetta.NewMemPoolAPIService(network, &api, rosettaLib)
	mempoolAPIController := server.NewMempoolAPIController(
		mempoolAPIService,
		asserter,
	)

	constructionAPIService := services.NewConstructionAPIService(network, &api)
	constructionAPIController := server.NewConstructionAPIController(
		constructionAPIService,
		asserter,
	)

	return server.NewRouter(accountAPIController, networkAPIController, blockAPIController,
		mempoolAPIController, constructionAPIController, callAPIController)
}

func startRosettaRPC(ctx context.Context, api api.FullNode) error {
	netName, _ := api.StateNetworkName(ctx)
	network := &types.NetworkIdentifier{
		Blockchain: BlockchainName,
		Network:    string(netName),
	}

	// The asserter automatically rejects incorrectly formatted
	// requests.
	asserter, err := rosettaAsserter.NewServer(
		rosetta.GetSupportedOpList(),
		true,
		[]*types.NetworkIdentifier{network},
		[]string{call.StateComputeCall},
		false,
		"",
	)
	if err != nil {
		rosetta.Logger.Fatal(err)
	}

	// Create instance of RosettaFilecoinLib for current network
	r := rosettaFilecoinLib.NewRosettaConstructionFilecoin(api)

	// Build trace retriever
	retriever := tools.NewTraceRetriever(
		viper.GetBool("use_cached_traces"),
		viper.GetString("trace_bucket"),
		data_store.DataStoreConfig{
			Url:      viper.GetString("data_store.url"),
			User:     viper.GetString("data_store.user"),
			Password: viper.GetString("data_store.password"),
			Service:  data_store.S3Storage,
		},
	)

	router := newBlockchainRouter(network, asserter, api, retriever, r)
	loggedRouter := server.LoggerMiddleware(router)
	corsRouter := server.CorsMiddleware(loggedRouter)
	server := &http.Server{Addr: fmt.Sprintf(":%d", ServerPort), Handler: corsRouter} //nolint

	sigCh := make(chan os.Signal, 2)

	go func() {
		<-sigCh
		rosetta.Logger.Warn("Shutting down rosetta...")

		err = server.Shutdown(context.TODO())
		if err != nil {
			rosetta.Logger.Error(err)
		} else {
			rosetta.Logger.Warn("Graceful shutdown of rosetta successful")
		}
	}()

	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	rosetta.Logger.Infof("Rosetta listening on port %d\n", ServerPort)
	return server.ListenAndServe()
}

func connectAPI(addr string, token string) (api.FullNode, jsonrpc.ClientCloser, error) { // nolint
	lotusAPI, clientCloser, err := getFullNodeAPI(addr, token)
	if err != nil {
		rosetta.Logger.Errorf("Error %s\n", err)
		return nil, nil, err
	}

	networkName, err := lotusAPI.StateNetworkName(context.Background())
	if err != nil {
		rosetta.Logger.Warn("Could not get Lotus network name!")
	}

	tools.NetworkName = string(networkName)

	version, err := lotusAPI.Version(context.Background())
	if err != nil {
		rosetta.Logger.Warn("Could not get Lotus api version!")
	}

	rosetta.Logger.Infof("Connected to Lotus node version: %s | Network: %s ", version.String(), tools.NetworkName)

	return lotusAPI, clientCloser, nil
}

func main() {
	startLogger("info")
	logVersionsInfo()

	addr := os.Getenv("LOTUS_RPC_URL")
	token := os.Getenv("LOTUS_RPC_TOKEN")

	rosetta.Logger.Info("Starting Rosetta Proxy")
	rosetta.Logger.Infof("LOTUS_RPC_URL: %s", addr)

	viper.SetConfigName("config")
	viper.AddConfigPath("/")
	viper.AddConfigPath(".")
	viper.SetDefault("use_cached_traces", false)

	var lotusAPI api.FullNode
	var clientCloser jsonrpc.ClientCloser // nolint
	var err error

	retryAttempts, _ := strconv.Atoi(rosetta.RetryConnectAttempts)

	for i := 1; i <= retryAttempts; i++ {
		lotusAPI, clientCloser, err = connectAPI(addr, token)
		if err == nil {
			break
		}
		rosetta.Logger.Errorf("Could not connect to api. Retrying attempt %d", i)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		rosetta.Logger.Fatalf("Connect to Lotus api gave up after %d attempts", retryAttempts)
		return
	}
	defer clientCloser()

	ctx := context.Background()
	err = startRosettaRPC(ctx, lotusAPI)
	if err != nil {
		rosetta.Logger.Info("Exit Rosetta rpc", err)
	}
}
