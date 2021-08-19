package main

import (
	"context"
	"fmt"
	"github.com/Zondax/zindexer/connections/data_store"
	"github.com/spf13/viper"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/services"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools/database"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools/parser"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	rosettaAsserter "github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	logging "github.com/ipfs/go-log"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
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
	traceRetriever *parser.TraceRetriever,
) http.Handler {
	accountAPIService := rosetta.NewAccountAPIService(network, &api)
	accountAPIController := server.NewAccountAPIController(
		accountAPIService,
		asserter,
	)

	networkAPIService := rosetta.NewNetworkAPIService(network, &api)
	networkAPIController := server.NewNetworkAPIController(
		networkAPIService,
		asserter,
	)

	blockAPIService := services.NewBlockAPIService(network, &api, traceRetriever)
	blockAPIController := server.NewBlockAPIController(
		blockAPIService,
		asserter,
	)

	mempoolAPIService := rosetta.NewMemPoolAPIService(network, &api)
	mempoolAPIController := server.NewMempoolAPIController(
		mempoolAPIService,
		asserter,
	)

	constructionAPIService := rosetta.NewConstructionAPIService(network, &api)
	constructionAPIController := server.NewConstructionAPIController(
		constructionAPIService,
		asserter,
	)

	return server.NewRouter(accountAPIController, networkAPIController,
		blockAPIController, mempoolAPIController, constructionAPIController)
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
		nil,
		false,
	)
	if err != nil {
		rosetta.Logger.Fatal(err)
	}

	// Build trace retriever
	retriever := parser.NewTraceRetriever(
		viper.GetBool("use_cached_traces"),
		viper.GetString("trace_bucket"),
		data_store.DataStoreConfig{
			Url:      viper.GetString("data_store.url"),
			User:     viper.GetString("data_store.user"),
			Password: viper.GetString("data_store.password"),
			Service:  data_store.MinIOStorage,
		},
	)

	router := newBlockchainRouter(network, asserter, api, retriever)
	loggedRouter := server.LoggerMiddleware(router)
	corsRouter := server.CorsMiddleware(loggedRouter)
	server := &http.Server{Addr: fmt.Sprintf(":%d", ServerPort), Handler: corsRouter}

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

func connectAPI(addr string, token string) (api.FullNode, jsonrpc.ClientCloser, error) {
	lotusAPI, clientCloser, err := getFullNodeAPI(addr, token)
	if err != nil {
		rosetta.Logger.Errorf("Error %s\n", err)
		return nil, nil, err
	}

	version, err := lotusAPI.Version(context.Background())
	if err != nil {
		rosetta.Logger.Warn("Could not get Lotus api version!")
	}

	rosetta.Logger.Info("Connected to Lotus version: ", version.String())

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
	viper.SetDefault("use_cached_traces", false)
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	var lotusAPI api.FullNode
	var clientCloser jsonrpc.ClientCloser

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

	database.SetupActorsDatabase(&lotusAPI)

	ctx := context.Background()
	err = startRosettaRPC(ctx, lotusAPI)
	if err != nil {
		rosetta.Logger.Info("Exit Rosetta rpc", err)
	}
}
