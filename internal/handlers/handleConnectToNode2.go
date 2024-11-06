package handlers

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net"

	"github.com/coldstar-507/utils"
)

const UPDATE_PREFIX byte = 0x10

func StartNodeConnServer() {
	listener, err := net.Listen("tcp", ":12000")
	utils.Panic(err, "StartNodeConnServer error on net.Listen")
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("error accepting connection:", err)
		} else {
			log.Println("new node connection:", conn.LocalAddr())
		}
		go handleNodeConn(conn)
	}
}

type ConnMessage struct {
	Payload []byte
	NodeId  *utils.NodeId
}

type man struct {
	subs            map[utils.NodeId]map[utils.Iddev]*nconn
	sub, unsub      chan *subreq
	clean           chan *nconn
	BroadcastUpdate chan *ConnMessage
}

var NodeConnMan = &man{
	subs:            make(map[utils.NodeId]map[utils.Iddev]*nconn),
	sub:             make(chan *subreq),
	unsub:           make(chan *subreq),
	clean:           make(chan *nconn),
	BroadcastUpdate: make(chan *ConnMessage),
}

type nconn struct {
	sess  int64
	iddev *utils.Iddev
	conn  net.Conn
	subs  []utils.NodeId
	res   chan struct{}
}

type subreq struct {
	nodeId *utils.NodeId
	nc     *nconn
}

func readReq(rdr io.Reader, connect *bool, buf *utils.NodeId, ts *int64) error {
	err0 := binary.Read(rdr, binary.BigEndian, connect)
	_, err1 := rdr.Read(buf[:])
	err2 := binary.Read(rdr, binary.BigEndian, ts)
	return errors.Join(err0, err1, err2)
}

func (nc *nconn) readFromConn() {
	log.Printf("client=%x listening for node connections\n", nc.iddev[:])
	defer log.Printf("client=%x done for node connections, closing thread\n", nc.iddev[:])

	var (
		connect bool
		nodeId  = utils.NodeId{}
		ts      int64
		comp    = func(a *utils.NodeId, b utils.NodeId) bool { return *a == b }
	)

	for {
		if err := readReq(nc.conn, &connect, &nodeId, &ts); err != nil {
			log.Printf("client=%x, error reading:\n%v\ndestroying\n",
				nc.iddev[:], err)
			NodeConnMan.clean <- nc
			return
		}

		if connect {
			// if not already connected, we fetch and write
			// the node if it has any update
			if !utils.ContainsWhere(&nodeId, nc.subs, comp) {
				strId := hex.EncodeToString(nodeId[:])
				newNode, _ := GetMongoNodeByIdAfter(strId, ts)
				if newNode != nil {
					binary.Write(nc.conn, binary.BigEndian, UPDATE_PREFIX)
					l := uint16(len(newNode))
					binary.Write(nc.conn, binary.BigEndian, l)
					nc.conn.Write(newNode)
				}
				nc.subs = append(nc.subs, nodeId)
				NodeConnMan.sub <- &subreq{nodeId: &nodeId, nc: nc}
				<-nc.res
			} else {
				log.Printf("nodeId=%x already in conns, for iddev=%x\n",
					nodeId[:], nc.iddev[:])
			}

		} else {
			if utils.ContainsWhere(&nodeId, nc.subs, comp) {
				nc.subs, _ = utils.Remove(nodeId, nc.subs)
				NodeConnMan.unsub <- &subreq{nodeId: &nodeId, nc: nc}
				<-nc.res
			} else {
				log.Printf("nodeId=%x not parts of conns for iddev=%x",
					nodeId[:], nc.iddev[:])
			}
		}
	}
}

func (m *man) Run() {
	for {
		select {
		case msg := <-m.BroadcastUpdate:
			log.Printf("Broadcast request for nodeId=%x\n", msg.NodeId[:])
			// log.Println("Subs:", m.subs)
			// log.Println("Raw nodeId:", *msg.NodeId)
			if subs := m.subs[*msg.NodeId]; subs != nil {
				for _, x := range subs {
					log.Printf("broadcasting to %x\n", x.iddev[:])
					x.conn.Write(msg.Payload)
				}
			}

		case cl := <-m.clean:
			log.Printf("Clean request for iddev=%x\n", cl.iddev[:])
			for _, x := range cl.subs {
				if node := m.subs[x]; m != nil {
					if u := node[*cl.iddev]; u != nil && u.sess == cl.sess {
						log.Printf("Removing from %x\n", x[:])
						delete(node, *cl.iddev)
						if len(node) == 0 {
							log.Printf("Room for nodeId=%x"+
								"is empty, removing it", x)
							delete(m.subs, x)
						}
					}
				}
			}
			close(cl.res)
			cl.conn.Close()

		case sub := <-m.sub:
			log.Printf("Subreq from iddev=%x to nodeId=%x\n",
				sub.nc.iddev[:], sub.nodeId[:])

			if subs_ := m.subs[*sub.nodeId]; subs_ == nil {
				log.Printf("creating sub group for nodeId=%x\n", sub.nodeId[:])
				subs_ = make(map[utils.Iddev]*nconn)
				subs_[*sub.nc.iddev] = sub.nc
				m.subs[*sub.nodeId] = subs_
			} else if sub_ := subs_[*sub.nc.iddev]; sub_ != nil {
				log.Println("already connected")
				if sub_.sess < sub.nc.sess {
					log.Println("cur sess:%d\nnew sess:%d\n",
						sub_.sess, sub.nc.sess)
					log.Println("replacing with newer conn")
					// replace with new conn
					subs_[*sub.nc.iddev] = sub.nc
					// this will trigger clean up
					sub_.conn.Close()
				} else if sub_.sess == sub.nc.sess {
					log.Println("req is from cur conn, ignoring")
				} else {
					log.Println("req is from older conn, closing")
					// new req is older ... either rare or impossible
					// this will trigger cleanup
					sub.nc.conn.Close()
				}
			} else {
				log.Println("adding the conn")
				subs_[*sub.nc.iddev] = sub.nc
			}

			sub.nc.res <- struct{}{}

		case unsub := <-m.unsub:
			log.Printf("Unsubreq from iddev=%x to nodeId=%x\n",
				unsub.nc.iddev[:], unsub.nodeId[:])
			if subs_ := m.subs[*unsub.nodeId]; subs_ != nil {
				if sub := subs_[*unsub.nc.iddev]; sub != nil {
					if sub.sess == unsub.nc.sess {
						log.Println("unsubbing")
						delete(subs_, *unsub.nc.iddev)
						if len(subs_) == 0 {
							log.Println("empty subs, removing")
							delete(m.subs, *unsub.nodeId)
						}
					}
				}
			}
			unsub.nc.res <- struct{}{}
		}
	}
}

func handleNodeConn(conn net.Conn) {
	iddev := utils.Iddev{}

	if _, err := conn.Read(iddev[:]); err != nil {
		log.Println("HandleNodeConn error reading iddev:", err)
		conn.Close()
		return
	}

	log.Printf("New node conn iddev=%x\n", iddev[:])

	wc := &nconn{
		sess:  utils.MakeTimestamp(),
		iddev: &iddev,
		conn:  conn,
		subs:  make([]utils.NodeId, 0, 5),
		res:   make(chan struct{}),
	}

	go wc.readFromConn()
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
