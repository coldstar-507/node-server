package db_listener

import (
	// "bytes"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"log"
	"strings"

	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/node-server/internal/handlers"
	"github.com/coldstar-507/utils"
	"go.mongodb.org/mongo-driver/bson"

	// "go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/mongo"
)

func MongoNodeListener() {
	ctx := context.Background()
	stream, err := db.Nodes.Watch(ctx, mongo.Pipeline{{
		{Key: "$match", Value: bson.M{"operationType": "update"}},
		// {Key: "updateDescription.updatedFields.interests",
		// 	Value: bson.M{"$exists": false}},
	}})

	utils.Must(err)
	defer stream.Close(ctx)
	for stream.Next(ctx) {
		val := stream.Current
		strId := val.Lookup("documentKey", "_id").StringValue()
		nodeId := utils.NodeId{}
		strRdr := strings.NewReader(strId)
		if _, err := hex.NewDecoder(strRdr).Read(nodeId[:]); err != nil {
			log.Println("MongoNodeListener: error reading hex id:", err)
			continue
		}
		fields := val.Lookup("updateDescription", "updatedFields")
		log.Println("node updated fields:", fields.String())
		l := utils.RAW_NODE_ID_LEN + len(fields.Value)
		fullLen := 1 + 2 + l
		buf := bytes.NewBuffer(make([]byte, 0, fullLen))
		buf.WriteByte(handlers.UPDATE_PREFIX)
		binary.Write(buf, binary.BigEndian, uint16(l))
		buf.Write(nodeId[:])
		buf.Write(fields.Value)
		msg := &handlers.ConnMessage{Payload: buf.Bytes(), NodeId: &nodeId}
		handlers.NodeConnMan.BroadcastUpdate <- msg
	}
}

func MongoUserListener() {
	ctx := context.Background()
	stream, err := db.Users.Watch(ctx, mongo.Pipeline{{
		{Key: "$match", Value: bson.M{"operationType": "update"}},
	}})

	utils.Must(err)
	defer stream.Close(ctx)
	for stream.Next(ctx) {
		val := stream.Current
		fields := val.Lookup("updateDescription", "updatedFields")
		log.Println("user updated fields:", fields.String())

		strId := val.Lookup("documentKey", "_id").StringValue()
		nodeId := utils.NodeId{}
		strRdr := strings.NewReader(strId)
		if _, err := hex.NewDecoder(strRdr).Read(nodeId[:]); err != nil {
			log.Println("MongoNodeListener: error reading hex id:", err)
			continue
		}

		bb := make([]byte, 3+len(fields.Value))
		bb[0] = handlers.UPDATE_PREFIX
		binary.BigEndian.PutUint16(bb[1:], uint16(len(fields.Value)))
		copy(bb[3:], fields.Value)
		msg := &handlers.ConnMessage{Payload: bb, NodeId: &nodeId}
		handlers.UserConnMan.BroadcastUpdate <- msg
	}
}

// sql version

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
			log.Printf("NodeListener error getting node id=%s, %v\n",
				ntf.Payload, err)
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
			log.Printf("UserListener error getting node id=%s, %v\n",
				ntf.Payload, err)
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
