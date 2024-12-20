package db

import (
	"context"
	"errors"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"

	"github.com/coldstar-507/utils/utils"
	"github.com/jackc/pgx/v5/pgxpool"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const DATABASE_URL string = "postgresql://scott:kendie@localhost:5432/custom-back"

var (
	Messager *messaging.Client
	Pool     *pgxpool.Pool
	err      error
	Mongo    *mongo.Client
	dbOne    *mongo.Database
	Nodes    *mongo.Collection
	Users    *mongo.Collection
	Tags     *mongo.Collection
)

var nodeNotifier = `
CREATE OR REPLACE FUNCTION notify_node_update()
RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify('node_changes', OLD.id);
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;
`

var userNotifier = `
CREATE OR REPLACE FUNCTION notify_user_update()
RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify('user_changes', OLD.id);
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;
`

var userTrigger = `
CREATE OR REPLACE TRIGGER user_update AFTER UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION notify_user_update();
`

var nodeTrigger = `
CREATE OR REPLACE TRIGGER user_update AFTER UPDATE ON nodes
  FOR EACH ROW EXECUTE FUNCTION notify_node_update();
`

var userTable = `
CREATE TABLE IF NOT EXISTS users (
  "id" TEXT PRIMARY KEY,
  "tag" TEXT UNIQUE NOT NULL,
  "neuter" TEXT NOT NULL,
  "token" TEXT NOT NULL,
  "roots" TEXT[] NOT NULL,
  "images" TEXT[] NOT NULL,
  "videos" TEXT[] NOT NULL,
  "gifs" TEXT[] NOT NULL,
  "interests" TEXT[] NOT NULL,
  "gender" TEXT,
  "geohash" TEXT,
  "age" INTEGER
)
`
var nodeTable = `
CREATE TABLE IF NOT EXISTS nodes (
  "id" TEXT PRIMARY KEY,
  "tag" TEXT UNIQUE NOT NULL,
  "type" TEXT NOT NULL,
  "name" TEXT NOT NULL,
  "lastName" TEXT,
  "mediaId" TEXT,
  "mainDeviceId" TEXT,
  "ownerId" TEXT,
  "hashTree" TEXT,
  "isPublic" BOOLEAN,
  "children" TEXT[],
  "posts" TEXT[],
  "admins" TEXT[],
  "members" TEXT[],
  "neuter" TEXT,
  "location" TEXT   
)
`

func InitFirebaseMessager() {
	servAcc := os.Getenv("FIREBASE_CONFIG")
	opt := option.WithCredentialsFile(servAcc)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	utils.Fatal(err, "InitFirebaseMessager error creating firebase app")
	Messager, err = app.Messaging(context.Background())
	utils.Fatal(err, "InitFirebaseMessager error creating firebase messager")
}

func initChannels() {
	_, err0 := Pool.Exec(context.Background(), "LISTEN user_changes;")
	_, err1 := Pool.Exec(context.Background(), "LISTEN node_changes;")
	_, err2 := Pool.Exec(context.Background(), userNotifier)
	_, err3 := Pool.Exec(context.Background(), nodeNotifier)
	_, err4 := Pool.Exec(context.Background(), userTrigger)
	_, err5 := Pool.Exec(context.Background(), nodeTrigger)
	utils.Fatal(errors.Join(err0, err1, err2, err3, err4, err5), "Error initializing channels")
}

func initTables() {
	_, err0 := Pool.Exec(context.Background(), userTable)
	_, err1 := Pool.Exec(context.Background(), nodeTable)
	utils.Fatal(errors.Join(err0, err1), "Error initializing tables")
}

func Init() {
	log.Println("Initializing db")
	Pool, err = pgxpool.New(context.Background(), DATABASE_URL)
	// Pool.Exec(context.Background(), "drop table nodes; drop table users;")
	utils.Fatal(err, "Init error opening pool")
	initTables()
	initChannels()
}

func ShutDown() {
	log.Println("Shutting down db")
	Pool.Close()
}

// uri := "mongodb://172.18.0.2:27017,172.18.0.3:27017/?replicatSet=mongo-replicas"
// uri := "mongodb://mongo-node:27017,mongo-node1:27017/?replicatSet=mongo-replicas"
func InitMongo() {
	// TODO set in env
	uri := "mongodb://localhost:27100,localhost:27200"
	opt := options.Client().
		SetReadPreference(readpref.Nearest()).
		ApplyURI(uri)

	Mongo, err = mongo.Connect(context.TODO(), opt)
	utils.Must(err)
	dbOne = Mongo.Database("one")
	Nodes = dbOne.Collection("nodes")
	Users = dbOne.Collection("users")
	Tags = dbOne.Collection("tags")

	if names, err := Nodes.Indexes().CreateMany(context.TODO(), []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "gender", Value: 1},
				{Key: "geohash", Value: 1},
				{Key: "age", Value: 1},
				{Key: "interests", Value: 1},
			},
		},
	}); err != nil {
		log.Println("InitMongo error creating user indexes:", err)
	} else {
		log.Println("Nodes indexes:", names)
	}
}

func InitMongo2() {
	// TODO set in env
	uri := "mongodb://localhost:27100,localhost:27200"
	opt := options.Client().
		SetReadPreference(readpref.Nearest()).
		ApplyURI(uri)

	Mongo, err = mongo.Connect(context.TODO(), opt)
	utils.Must(err)
	dbOne = Mongo.Database("one")
	Nodes = dbOne.Collection("nodes")
	Tags = dbOne.Collection("tags")

	if names, err := Nodes.Indexes().CreateOne(context.TODO(), mongo.IndexModel{
		Keys: bson.D{
			{Key: "user.gender", Value: 1},
			{Key: "user.geohash", Value: 1},
			{Key: "user.age", Value: 1},
			{Key: "user.interests", Value: 1},
		},
	}); err != nil {
		log.Println("InitMongo error creating user indexes:", err)
	} else {
		log.Println("User indexes:", names)
	}
}

func ShutdownMongo() {
	if err := Mongo.Disconnect(context.TODO()); err != nil {
		panic(err)
	}
}
