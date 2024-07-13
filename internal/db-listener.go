package db_listener

import (
	"context"
	"encoding/binary"
	"log"

	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/node-server/internal/handlers"
	"github.com/coldstar-507/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func NodeListener() {
	poolconn, err := db.Pool.Acquire(context.Background())
	utils.Must(err)
	conn := poolconn.Hijack()
	defer conn.Close(context.Background())
	_, err = conn.Exec(context.Background(), "LISTEN node_changes")
	utils.Must(err)
	for {
		ntf, err := conn.WaitForNotification(context.Background())
		utils.NonFatal(err, "NodeListener notification error")
		if node, err := handlers.GetNodeById(ntf.Payload); err != nil {
			log.Printf("NodeListener error getting node id=%s, %v\n", ntf.Payload, err)
		} else {
			bb := make([]byte, 3+len(node))
			bb[0] = 0x00
			binary.BigEndian.PutUint16(bb[1:], uint16(len(node)))
			copy(bb[3:], node)
			msg := &handlers.ConnMsg{Id: ntf.Payload, Payload: bb}
			handlers.NodeConnectionManager.Broadcast <- msg
		}
	}
}

func UserListener() {
	poolconn, err := db.Pool.Acquire(context.Background())
	utils.Must(err)
	conn := poolconn.Hijack()
	defer conn.Close(context.Background())
	_, err = conn.Exec(context.Background(), "LISTEN user_changes")
	utils.Must(err)
	for {
		ntf, err := conn.WaitForNotification(context.Background())
		utils.NonFatal(err, "UserListener notification error")
		if node, err := handlers.GetNodeById(ntf.Payload); err != nil {
			log.Printf("UserListener error getting node id=%s, %v\n", ntf.Payload, err)
		} else {
			bb := make([]byte, 3+len(node))
			bb[0] = 0x00
			binary.BigEndian.PutUint16(bb[1:], uint16(len(node)))
			copy(bb[3:], node)
			msg := &handlers.ConnMsg{Id: ntf.Payload, Payload: bb}
			handlers.UserConnectionManager.Broadcast <- msg
		}
	}
}

func MongoNodeListener() {
	ctx := context.Background()
	stream, err := db.Nodes.Watch(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"operationType": "update"}}}})

	utils.Must(err)
	defer stream.Close(ctx)
	for stream.Next(ctx) {
		val := stream.Current
		id := val.Lookup("documentKey", "_id").StringValue()
		fields := val.Lookup("updateDescription", "updatedFields")
		log.Printf("updated fields: %v, t: %v\n", fields, fields.Type)
		bb := make([]byte, 3+len(fields.Value))
		bb[0] = 0x00
		binary.BigEndian.PutUint16(bb[1:], uint16(len(fields.Value)))
		copy(bb[3:], fields.Value)
		msg := &handlers.ConnMsg{Payload: bb, Id: id}
		handlers.NodeConnectionManager.Broadcast <- msg
	}
}

func MongoUserListener() {
	ctx := context.Background()
	stream, err := db.Users.Watch(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"operationType": "update"}}}})

	utils.Must(err)
	defer stream.Close(ctx)
	for stream.Next(ctx) {
		val := stream.Current
		fields := val.Lookup("updateDescription", "updatedFields")
		id := val.Lookup("documentKey", "_id").StringValue()
		bb := make([]byte, 3+len(fields.Value))
		bb[0] = 0x00
		binary.BigEndian.PutUint16(bb[1:], uint16(len(fields.Value)))
		copy(bb[3:], fields.Value)
		msg := &handlers.ConnMsg{Payload: bb, Id: id}
		handlers.UserConnectionManager.Broadcast <- msg
	}
}

// func MongoPublicListener() {
// 	ctx := context.Background()
// 	stream, err := db.Nodes.Watch(ctx, mongo.Pipeline{
// 		{{Key: "$match", Value: bson.M{"operationType": "update"}}},
// 		{{Key: "$match", Value: bson.M{"public": bson.M{"$exists": true}}}},
// 		{{Key: "$project", Value: bson.M{"public": 1}}},
// 		{{Key: "$addFields", Value: bson.M{"root": "$root"}}},
// 	})
// 	utils.Must(err)
// 	defer stream.Close(ctx)

// 	buf := new(bytes.Buffer)
// 	for stream.Next(ctx) {
// 		val := stream.Current
// 		fields := val.Lookup("updateDescription", "updatedFields")
// 		root := val.Lookup("root").String()
// 		buf.Write(fields.Value)
// 		payload := make([]byte, len(buf.Bytes()))
// 		copy(payload, buf.Bytes())
// 		handlers.PubConnManager.Broadcast <- &handlers.ConnMsg{Payload: payload, Root: root}
// 		buf.Reset()
// 	}
// }

// func MongoPrivateListener() {
// 	ctx := context.Background()
// 	stream, err := db.Nodes.Watch(ctx, mongo.Pipeline{
// 		{{Key: "$match", Value: bson.M{"operationType": "update"}}},
// 		{{Key: "$match", Value: bson.M{"private": bson.M{"$exists": true}}}},
// 		{{Key: "$addFields", Value: bson.M{"root": "$root"}}},
// 		{{Key: "$project", Value: bson.M{"public": 1}}}})

// 	utils.Must(err)
// 	defer stream.Close(ctx)

// 	buf := new(bytes.Buffer)
// 	for stream.Next(ctx) {
// 		val := stream.Current
// 		fields := val.Lookup("updateDescription", "updatedFields")
// 		root := val.Lookup("root").String()
// 		buf.Write(fields.Value)
// 		payload := make([]byte, len(buf.Bytes()))
// 		copy(payload, buf.Bytes())
// 		handlers.PrivConnManager.Broadcast <- &handlers.ConnMsg{Payload: payload, Root: root}
// 		buf.Reset()
// 	}
// }

// func NodeChangeListener() {
// 	acq, err := db.Pool.Acquire(context.Background())
// 	if err != nil {
// 		log.Fatalln("NodeChangeListener error aqcuireing conn:", err)
// 	}
// 	conn := acq.Hijack()
// 	for {
// 		ntf, err := conn.WaitForNotification(context.Background())
// 		if err != nil {
// 			log.Println("NodeChangeListener error waiting for notification:", err)
// 			continue
// 		}
// 		root := ntf.Payload
// 		node, err := handlers.GetNode(root)
// 		if err != nil {
// 			log.Println("NodeChangeListener error getting updated node:", err)
// 			continue
// 		}
// 		handlers.PubConnManager.Broadcast <- &handlers.ConnMsg{Payload: node, Root: root}
// 	}
// }
