package main

import (
	"log"
	"net/http"

	db_listener "github.com/coldstar-507/node-server/internal"
	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/node-server/internal/handlers"
	"github.com/coldstar-507/utils"
)

var (
	ip    string            = "localhost"
	st    utils.SERVER_TYPE = utils.NODE_ROUTER
	place uint16            = 0x0000
)

func main() {
	db.InitMongo()
	log.Println("Mongo client initialized")
	defer db.ShutdownMongo()

	go handlers.NodeConnMan.Run()
	go handlers.StartNodeConnServer()
	go db_listener.MongoNodeListener()

	go handlers.UserConnMan.Run()
	go handlers.StartUserConnServer()
	go db_listener.MongoUserListener()

	utils.InitLocalRouter(ip, st, place)
	go utils.LocalRouter.Run()
	log.Println("LocalRouter is running")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", utils.HandlePing)
	mux.HandleFunc("GET /route-scores", utils.HandleScoreRequest)

	mux.HandleFunc("GET /valid-tag/{tag}", handlers.HandleValidTag)
	mux.HandleFunc("POST /init", handlers.HandleInit)
	mux.HandleFunc("POST /boost", handlers.HandleBoostRequest)

	mux.HandleFunc("GET /node/tag/{tag}", handlers.HandleGetNodeByTag)
	mux.HandleFunc("GET /node/id/{id}", handlers.HandleGetNodeById)
	// can mask only valid updatable fields
	// have it sent a secret with it? secret must be hidden from client
	mux.HandleFunc("PATCH /node/id/{id}", handlers.HandleUpdateNode)

	mux.HandleFunc("GET /nodes/tags/{tags}", handlers.HandleGetNodesByTags)
	mux.HandleFunc("GET /nodes/ids/{ids}", handlers.HandleGetNodesByIds)

	mux.HandleFunc("GET /all-nodes", handlers.HandleAllNodes)
	mux.HandleFunc("GET /all-users", handlers.HandleAllUsers)
	mux.HandleFunc("GET /all-tags", handlers.HandleAllTags)
	mux.HandleFunc("GET /pretty-user/tag/{tag}", handlers.HandleGetPrettyUserByTag)
	mux.HandleFunc("GET /pretty-user/id/{id}", handlers.HandleGetPrettyUserById)

	mux.HandleFunc("GET /pretty-node/tag/{tag}", handlers.HandleGetPrettyNodeByTag)
	mux.HandleFunc("GET /pretty-node/id/{id}", handlers.HandleGetPrettyNodeById)

	mux.HandleFunc("POST /create-group", handlers.HandleCreateGroup)
	mux.HandleFunc("GET /add-to-group/{id}/{ids}", handlers.HandleAddToGroup)

	mux.HandleFunc("GET /push-medias/{id}/{type}/{medias}", handlers.HandlePushMedias)
	mux.HandleFunc("GET /push-nfts/{id}/{ids}", handlers.HandlePushNfts)
	mux.HandleFunc("GET /push-root/{root}/{ids}", handlers.HandlePushRoot)
	mux.HandleFunc("GET /update-interests/{id}/{interests}", handlers.HandleUpdateInterests)

	mux.HandleFunc("GET /node-connection/{id}/{iddev}", handlers.HandleNodeConnection)
	mux.HandleFunc("GET /user-connection/{id}/{iddev}", handlers.HandleUserConnection)

	server := utils.ApplyMiddlewares(mux,
		utils.StatusLogger,
		// utils.HttpLogging
	)

	addr := "0.0.0.0:8083"
	log.Println("Starting http node-server on", addr)
	err := http.ListenAndServe(addr, server)
	utils.NonFatal(err, "ERROR http.ListenAndServe error")
}
