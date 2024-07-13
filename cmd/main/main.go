package main

import (
	"log"
	"net/http"

	db_listener "github.com/coldstar-507/node-server/internal"
	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/node-server/internal/handlers"
	"github.com/coldstar-507/utils"
)

func main() {
	db.InitMongo()
	log.Println("Mongo client initialized")
	defer db.ShutdownMongo()
	// db.Init()
	// log.Println("Gres client initialized")
	// defer db.ShutDown()

	go handlers.UserConnectionManager.Run()
	log.Println("User connection manager is running")
	go handlers.NodeConnectionManager.Run()
	log.Println("Node connection manager is running")

	go db_listener.MongoUserListener()
	// go db_listener.UserListener()
	log.Println("User listener is running")
	go db_listener.MongoNodeListener()
	// go db_listener.NodeListener()
	log.Println("Node listener is running")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", handlers.HandlePing)
	mux.HandleFunc("GET /valid-username/{username}", handlers.HandleValidUsername)
	mux.HandleFunc("POST /init", handlers.HandleInit)
	mux.HandleFunc("POST /boost", handlers.HandleBoostRequest)

	mux.HandleFunc("POST /node", handlers.HandleUploadNode)
	mux.HandleFunc("GET /node/tag/{tag}", handlers.HandleGetNodeByTag)
	mux.HandleFunc("GET /node/id/{id}", handlers.HandleGetNodeById)
	mux.HandleFunc("PATCH /node/id/{id}", handlers.HandleUpdateNode)
	mux.HandleFunc("DELETE /node/id/{id}", handlers.HandleDeleteNode)
	mux.HandleFunc("GET /nodes/tags/{tags}", handlers.HandleGetNodesByTags)
	mux.HandleFunc("GET /nodes/ids/{ids}", handlers.HandleGetNodesByIds)

	// mux.HandleFunc("GET /push-chat-id/{id}/{targets}", handlers.HandlePushChatId)
	// mux.HandleFunc("GET /push-medias/{id}/{medias}", handlers.HandlePushMedias)
	mux.HandleFunc("GET /node-connection/{id}/{iddev}", handlers.HandleNodeConnection)
	mux.HandleFunc("GET /user-connection/{id}/{iddev}", handlers.HandleUserConnection)

	server := utils.ApplyMiddlewares(mux,
		utils.StatusLogger,
		utils.HttpLogging)

	log.Println("Starting http node-server on port 8080")
	err := http.ListenAndServe("localhost:8080", server)
	utils.NonFatal(err, "ERROR http.ListenAndServe error")
}
