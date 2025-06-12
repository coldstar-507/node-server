package main

import (
	"log"
	"net/http"

	db_listener "github.com/coldstar-507/node-server/internal"
	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/node-server/internal/handlers"
	"github.com/coldstar-507/router/router_utils"
	"github.com/coldstar-507/utils/http_utils"
	"github.com/coldstar-507/utils/utils"
)

var (
	ip         string                   = "localhost"
	place      router_utils.SERVER_NUMB = 0000
	routerType router_utils.ROUTER_TYPE = router_utils.NODE_ROUTER
)

func main() {
	db.InitMongo()
	log.Println("Mongo client initialized")
	defer db.ShutdownMongo()

	// go handlers.NodeConnMan.Run()
	go handlers.StartNodeConnServer3()
	go db_listener.MongoNodeListener()

	// go handlers.UserConnMan.Run()
	// go handlers.StartUserConnServer()
	// go db_listener.MongoUserListener()

	router_utils.InitLocalServer(ip, place, routerType)
	go router_utils.LocalServer.Run()
	log.Println("LocalRouter is running")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", router_utils.HandlePing)
	mux.HandleFunc("GET /route-scores", router_utils.HandleScoreRequest)
	mux.HandleFunc("GET /local-router", router_utils.HandleServerStatus)
	mux.HandleFunc("GET /full-router", router_utils.HandleRouterStatus)

	mux.HandleFunc("GET /valid-tag/{tag}", handlers.HandleValidTag)
	mux.HandleFunc("POST /init", handlers.HandleInit)
	mux.HandleFunc("POST /boost", handlers.HandleBoostRequest)

	mux.HandleFunc("GET /account-state/{phone}/{nodeId}", handlers.HandleAccountState)
	mux.HandleFunc("GET /create-account/{phone}/{countryCode}/{nodeId}",
		handlers.HandleCreateAccount)

	// mux.HandleFunc("PATCH /user/id/{id}", handlers.HandleUpdateUserById)

	mux.HandleFunc("GET /node/tag/{tag}", handlers.HandleGetNodeByTag)
	mux.HandleFunc("GET /node/id/{id}", handlers.HandleGetNodeById)
	// can mask only valid updatable fields
	// have it sent a secret with it? secret must be hidden from client
	mux.HandleFunc("PATCH /node/id/{id}/{ts}", handlers.HandleUpdateNodeById)
	mux.HandleFunc("POST /node", handlers.HandleUploadNode)
	// mux.HandleFunc("DELETE /node/id/{id}", handlers.HandleDeleteNode)
	mux.HandleFunc("DELETE /node/tag/{tag}", handlers.HandleDeleteNodeByTag)

	mux.HandleFunc("GET /nodes/tags/{tags}", handlers.HandleGetNodesByTags)
	mux.HandleFunc("GET /nodes/ids/{ids}", handlers.HandleGetNodesByIds)

	mux.HandleFunc("GET /all-nodes", handlers.HandleAllNodes)
	// mux.HandleFunc("GET /all-users", handlers.HandleAllUsers)
	mux.HandleFunc("GET /all-tags", handlers.HandleAllTags)
	// mux.HandleFunc("GET /pretty-user/tag/{tag}", handlers.HandleGetPrettyUserByTag)
	// mux.HandleFunc("GET /pretty-user/id/{id}", handlers.HandleGetPrettyUserById)

	mux.HandleFunc("GET /all-accounts", handlers.HandleAllAccounts)

	mux.HandleFunc("GET /pretty-node/tag/{tag}", handlers.HandleGetPrettyNodeByTag)
	mux.HandleFunc("GET /pretty-node/id/{id}", handlers.HandleGetPrettyNodeById)

	mux.HandleFunc("POST /create-group", handlers.HandleCreateGroup)
	mux.HandleFunc("GET /add-to-group/{id}/{ids}", handlers.HandleAddToGroup)

	// mux.HandleFunc("GET /get-root-for/{id1}/{id2}/{chatPlace}", handlers.HandleGetRootFor)

	// mux.HandleFunc("GET /push-medias/{id}/{type}/{medias}", handlers.HandlePushMedias)
	// mux.HandleFunc("GET /push-nfts/{id}/{ids}", handlers.HandlePushNfts)
	// mux.HandleFunc("GET /push-root/{root}/{ids}", handlers.HandlePushRoot)

	// mux.HandleFunc("GET /update-interests/{id}/{interests}", handlers.HandleUpdateInterests)

	// mux.HandleFunc("GET /node-connection/{id}/{iddev}", handlers.HandleNodeConnection)
	// mux.HandleFunc("GET /user-connection/{id}/{iddev}", handlers.HandleUserConnection)

	// handlers.DeleteAllAccounts()

	server := http_utils.ApplyMiddlewares(mux, http_utils.StatusLogger)

	addr := "0.0.0.0:8083"
	log.Println("Starting http node-server on", addr)
	err := http.ListenAndServe(addr, server)
	utils.NonFatal(err, "ERROR http.ListenAndServe error")
}
