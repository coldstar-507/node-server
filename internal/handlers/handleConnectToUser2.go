package handlers

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"log"
	"net"

	"github.com/coldstar-507/utils"
)

func StartUserConnServer() {
	listener, err := net.Listen("tcp", ":12001")
	utils.Panic(err, "StartUserConnServer error on net.Listen")
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("error accepting connection:", err)
		} else {
			log.Println("new user connection:", conn.LocalAddr())
		}
		go handleUserConn(conn)
	}
}

type uman struct {
	subs            map[utils.NodeId]map[utils.Dev]*uconn
	clean           chan *uconn
	new             chan *uconn
	BroadcastUpdate chan *ConnMessage
}

type uconn struct {
	sess   int64
	dev    *utils.Dev
	nodeId *utils.NodeId
	conn   net.Conn
}

var UserConnMan = &uman{
	subs:            make(map[utils.NodeId]map[utils.Dev]*uconn),
	clean:           make(chan *uconn),
	new:             make(chan *uconn),
	BroadcastUpdate: make(chan *ConnMessage),
}

func readUserConnReq(r io.Reader, devBuf *utils.Dev, nodeIdBuf *utils.NodeId, ts *int64) error {
	_, err := r.Read(devBuf[:])
	if err != nil {
		return err
	}
	_, err = r.Read(nodeIdBuf[:])
	if err != nil {
		return err
	}
	binary.Read(r, binary.BigEndian, ts)
	return nil
}

func (m *uman) Run() {
	for {
		select {
		case msg := <-m.BroadcastUpdate:
			log.Printf("Broadcast request for user id=%x\n", msg.NodeId[:])
			if subs := m.subs[*msg.NodeId]; subs != nil {
				for _, x := range subs {
					log.Printf("broadcasting to dev=%x\n", x.dev[:])
					x.conn.Write(msg.Payload)
				}
			}

		case cl := <-m.clean:
			log.Printf("Clean request for dev=%x\n", cl.dev[:])
			if x := m.subs[*cl.nodeId]; x != nil {
				if y := x[*cl.dev]; y != nil && y.sess == cl.sess {
					log.Printf("Removing dev=%x for nodeId=%x\n",
						cl.dev[:], cl.nodeId[:])
					delete(x, *cl.dev)
					if len(x) == 0 {
						log.Printf("nodeId=%x is empty, removing\n",
							cl.nodeId[:])
						delete(m.subs, *cl.nodeId)
					}
				}
			}
			cl.conn.Close()

		case n := <-m.new:
			log.Printf("New request for dev=%x for nodeId=%x\n",
				n.dev[:], n.nodeId[:])
			if x := m.subs[*n.nodeId]; x == nil {
				log.Println("No map, creating it")
				x = map[utils.Dev]*uconn{*n.dev: n}
				m.subs[*n.nodeId] = x
			} else {
				log.Println("Adding to map")
				x[*n.dev] = n
			}

		}
	}
}

func handleUserConn(conn net.Conn) {
	var (
		dev     = utils.Dev{}
		nodeId  = utils.NodeId{}
		ts      int64
		discBuf = make([]byte, 1)
	)

	if err := readUserConnReq(conn, &dev, &nodeId, &ts); err != nil {
		log.Println("handleUserConn: readuserConnReq error:", err)
		conn.Close()
		return
	}

	strId := hex.EncodeToString(nodeId[:])
	if newUser, _ := GetMongoUserByIdAfter(strId, ts); newUser != nil {
		binary.Write(conn, binary.BigEndian, uint16(len(newUser)))
		conn.Write(newUser)
	}

	uc := &uconn{
		sess:   utils.MakeTimestamp(),
		dev:    &dev,
		nodeId: &nodeId,
		conn:   conn,
	}

	UserConnMan.new <- uc
	// this will block until disconnection from user
	conn.Read(discBuf)
	UserConnMan.clean <- uc

}

// var up = websocket.Upgrader{
// 	ReadBufferSize:  1024,
// 	WriteBufferSize: 1024,
// }

// func HandleNodeConn(w http.ResponseWriter, r *http.Request) {
// 	defer r.Body.Close()
// 	iddev := utils.Iddev{}
// 	if wsconn, err := up.Upgrade(w, r, nil); err != nil {
// 		w.WriteHeader(500)
// 		log.Println("HandleNodeConn error upgrading:", err)
// 	} else if _, r, err := wsconn.NextReader(); err != nil {
// 		w.WriteHeader(501)
// 		log.Println("HandleNodeConn error getting first reader:", err)
// 	} else if _, err = r.Read(iddev[:]); err != nil {
// 		w.WriteHeader(502)
// 		log.Println("HandleNodeConn error reading into iddev:", err)
// 	} else {
// 		wc := &wconn{
// 			sess:  utils.MakeTimestamp(),
// 			iddev: &iddev,
// 			conn:  wsconn,
// 			subs:  make([]*utils.NodeId, 0, 5),
// 			res:   make(chan struct{}),
// 		}

// 		go wc.read()
// 	}
// }
