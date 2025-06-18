package db_listener

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"log"
	"strings"

	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/node-server/internal/handlers"
	"github.com/coldstar-507/utils2"
	// "github.com/coldstar-507/utils/id_utils"
	// "github.com/coldstar-507/utils/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func MongoNodeListener() {
	ctx := context.Background()
	stream, err := db.Nodes.Watch(ctx, mongo.Pipeline{{
		{Key: "$match", Value: bson.M{"operationType": "update"}},
		// {Key: "updateDescription.updatedFields.interests",
		// 	Value: bson.M{"$exists": false}},
	}})

	utils2.Must(err)
	defer stream.Close(ctx)
	for stream.Next(ctx) {
		val := stream.Current
		strId := val.Lookup("documentKey", "_id").StringValue()
		nodeId := utils2.NodeId{}
		strRdr := strings.NewReader(strId)
		if _, err := hex.NewDecoder(strRdr).Read(nodeId[:]); err != nil {
			log.Println("MongoNodeListener: error reading hex id:", err)
			continue
		}
		fields := val.Lookup("updateDescription", "updatedFields")
		log.Println("node updated fields:", fields.String())
		fullLen := 1 + utils2.RAW_NODE_ID_LEN + 2 + len(fields.Value)
		buf := bytes.NewBuffer(make([]byte, 0, fullLen))
		buf.WriteByte(handlers.UPDATE_PREFIX)
		buf.Write(nodeId[:])
		binary.Write(buf, binary.BigEndian, uint16(len(fields.Value)))
		buf.Write(fields.Value)

		handlers.NodeBroadcast2(nodeId, buf.Bytes())

		// msg := &handlers.ConnMessage{Payload: buf.Bytes(), NodeId: &nodeId}
		// handlers.NCMS.GetMan(&nodeId).BroadcastUpdate <- msg
		// handlers.NodeConnMan.BroadcastUpdate <- msg
	}
}

// func MongoUserListener() {
// 	ctx := context.Background()
// 	stream, err := db.Users.Watch(ctx, mongo.Pipeline{{
// 		{Key: "$match", Value: bson.M{"operationType": "update"}},
// 	}})

// 	utils2.Must(err)
// 	defer stream.Close(ctx)
// 	for stream.Next(ctx) {
// 		val := stream.Current
// 		fields := val.Lookup("updateDescription", "updatedFields")
// 		log.Println("user updated fields:", fields.String())

// 		strId := val.Lookup("documentKey", "_id").StringValue()
// 		nodeId := utils2.NodeId{}
// 		strRdr := strings.NewReader(strId)
// 		if _, err := hex.NewDecoder(strRdr).Read(nodeId[:]); err != nil {
// 			log.Println("MongoNodeListener: error reading hex id:", err)
// 			continue
// 		}

// 		bb := make([]byte, 3+len(fields.Value))
// 		bb[0] = handlers.UPDATE_PREFIX
// 		binary.BigEndian.PutUint16(bb[1:], uint16(len(fields.Value)))
// 		copy(bb[3:], fields.Value)
// 		msg := &handlers.ConnMessage{Payload: bb, NodeId: &nodeId}
// 		handlers.UserConnMan.BroadcastUpdate <- msg
// 	}
// }

// sql version

// func NodeListener() {
// 	poolconn, err := db.Pool.Acquire(context.Background())
// 	utils2.Must(err)
// 	conn := poolconn.Hijack()
// 	defer conn.Close(context.Background())
// 	_, err = conn.Exec(context.Background(), "LISTEN node_changes")
// 	utils2.Must(err)
// 	for {
// 		ntf, err := conn.WaitForNotification(context.Background())
// 		utils2.NonFatal(err, "NodeListener notification error")
// 		if node, err := handlers.GetNodeById(ntf.Payload); err != nil {
// 			log.Printf("NodeListener error getting node id=%s, %v\n",
// 				ntf.Payload, err)
// 		} else {
// 			bb := make([]byte, 3+len(node))
// 			bb[0] = 0x00
// 			binary.BigEndian.PutUint16(bb[1:], uint16(len(node)))
// 			copy(bb[3:], node)
// 			msg := &handlers.ConnMsg{Id: ntf.Payload, Payload: bb}
// 			handlers.NodeConnectionManager.Broadcast <- msg
// 		}
// 	}
// }

// func UserListener() {
// 	poolconn, err := db.Pool.Acquire(context.Background())
// 	utils2.Must(err)
// 	conn := poolconn.Hijack()
// 	defer conn.Close(context.Background())
// 	_, err = conn.Exec(context.Background(), "LISTEN user_changes")
// 	utils2.Must(err)
// 	for {
// 		ntf, err := conn.WaitForNotification(context.Background())
// 		utils2.NonFatal(err, "UserListener notification error")
// 		if node, err := handlers.GetNodeById(ntf.Payload); err != nil {
// 			log.Printf("UserListener error getting node id=%s, %v\n",
// 				ntf.Payload, err)
// 		} else {
// 			bb := make([]byte, 3+len(node))
// 			bb[0] = 0x00
// 			binary.BigEndian.PutUint16(bb[1:], uint16(len(node)))
// 			copy(bb[3:], node)
// 			msg := &handlers.ConnMsg{Id: ntf.Payload, Payload: bb}
// 			handlers.UserConnectionManager.Broadcast <- msg
// 		}
// 	}
// }
