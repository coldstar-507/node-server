package handlers

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/coldstar-507/utils/id_utils"
	"github.com/coldstar-507/utils/utils"
	"go.mongodb.org/mongo-driver/bson"
)

const UPDATE_PREFIX byte = 0x10
const heartbeat byte = 0x99

var NCMS *nodeConnsManagers
var cm = &connMan{
	conns: make(map[id_utils.Iddev_]*nconn),
	addCh: make(chan *nconn),
	delCh: make(chan *nconn),
}

type connMan struct {
	conns        map[id_utils.Iddev_]*nconn
	addCh, delCh chan *nconn
}

func (cm *connMan) run() {
	log.Println("cm: run()")
	for {
		select {
		case a := <-cm.addCh:
			log.Printf("cm: addReq:%x\n", *a.iddev)
			if co := cm.conns[*a.iddev]; co != nil {
				log.Printf("cm: co already there for %x\n", *a.iddev)
				if a.sess > co.sess {
					log.Printf("cm: newer sess for %x\n", *a.iddev)
					co.conn.C.Close()
					delete(cm.conns, *co.iddev)
					cm.conns[*a.iddev] = a
					a.res <- nil
				} else {
					log.Printf("cm: older sess for %x\n", *a.iddev)
					a.conn.C.Close()
					a.res <- errors.New("older session")
				}
			} else {
				log.Printf("cm: adding conn for %x\n", *a.iddev)
				cm.conns[*a.iddev] = a
				a.res <- nil
			}
		case r := <-cm.delCh:
			log.Printf("cm: delReq for %x\n", *r.iddev)
			if co := cm.conns[*r.iddev]; co != nil {
				log.Printf("cm: co is there for %x\n", *r.iddev)
				co.conn.C.Close()
				delete(cm.conns, *co.iddev)
			}
		}
	}
}

func StartNodeConnServer() {
	listener, err := net.Listen("tcp", ":12000")
	utils.Panic(err, "StartNodeConnServer: error on net.Listen")
	defer listener.Close()
	n, err := strconv.Atoi(os.Getenv("N_NODE_MANAGERS"))
	utils.Panic(err, "StartNodeConnServer: undefined N_NODE_MANAGERS")
	initNodeConnsManagers(uint32(n))
	go cm.run()

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
	NodeId  *id_utils.NodeId
}

type man struct {
	i               uint32
	subs            map[id_utils.NodeId]map[id_utils.Iddev_]*nconn
	sub, unsub      chan *subreq
	clean           chan *nconn
	BroadcastUpdate chan *ConnMessage
}

func createMan(r uint32) *man {
	return &man{
		i:               r,
		subs:            make(map[id_utils.NodeId]map[id_utils.Iddev_]*nconn),
		sub:             make(chan *subreq),
		unsub:           make(chan *subreq),
		clean:           make(chan *nconn),
		BroadcastUpdate: make(chan *ConnMessage),
	}
}

type nconn struct {
	sess  int64
	iddev *id_utils.Iddev_
	conn  *utils.ClientConn
	// conn  net.Conn
	subs []id_utils.NodeId
	res  chan error
}

type subreq struct {
	nodeId *id_utils.NodeId
	nc     *nconn
	res    chan struct{}
}

type nodeConnsManagers struct {
	nMan uint32
	mans []*man
}

func initNodeConnsManagers(nMans uint32) {
	NCMS = &nodeConnsManagers{
		nMan: nMans,
		mans: make([]*man, nMans),
	}
	for i := range NCMS.nMan {
		NCMS.mans[i] = createMan(i)
		go NCMS.mans[i].Run()
	}
}

func (ncm *nodeConnsManagers) GetMan(nodeId *id_utils.NodeId) *man {
	h := fnv.New32()
	h.Write(nodeId[:])
	i := h.Sum32() % ncm.nMan
	return ncm.mans[i]
}

func (nc *nconn) readFromConn() {
	log.Printf("client=%x, sess=%d listening for node connections\n", nc.iddev[:], nc.sess)
	defer log.Printf("client=%x, sess=%d done for node connections, closing thread\n",
		nc.iddev[:], nc.sess)

	var (
		err     error
		ch      = make(chan error)
		connect bool
		nodeId  = id_utils.NodeId{}
		ts      int64
		comp    = func(a *id_utils.NodeId, b id_utils.NodeId) bool { return *a == b }
	)

	go func() {
		ticker := time.NewTicker(time.Second * 20)
		defer log.Printf("killed heartbeater for %x, sess=%d\n", *nc.iddev, nc.sess)
		defer ticker.Stop()
		var heartbeatErr error
		for {
			<-ticker.C
			heartbeatErr = nc.conn.WriteBin(heartbeat)
			if heartbeatErr != nil {
				nc.conn.C.Close()
				return
			}
			log.Printf("client=%x, sess=%d, heartbeat", *nc.iddev, nc.sess)
		}
	}()

	for {

		err = utils.ReadBin(nc.conn.C, &connect, nodeId[:], &ts)
		if err != nil {
			close(ch)
			// ticker.Stop()
			for _, sub := range nc.subs {
				NCMS.GetMan(&sub).clean <- nc
				// NodeConnMan.clean <- nc
			}
			cm.delCh <- nc
			return
		}

		if connect {
			// if not already connected, we fetch and write
			// the node if it has any update
			if !utils.ContainsWhere(&nodeId, nc.subs, comp) {
				nc.subs = append(nc.subs, nodeId)
				res := make(chan struct{})
				req := &subreq{res: res, nodeId: &nodeId, nc: nc}
				NCMS.GetMan(&nodeId).sub <- req
				// NodeConnMan.sub <- &subreq{nodeId: &nodeId, nc: nc}
				<-res
				close(res)

				strId := hex.EncodeToString(nodeId[:])
				newNode, _ := GetMongoNodeByIdAfter(strId, ts)
				fmt.Printf("readFromConn: GetMongogoNodeByIdAfter(%s, %d)\n",
					strId, ts)
				if newNode != nil {
					bsonId := bson.Raw(newNode).Lookup("_id")
					fmt.Printf("readFromConn: got the node: id=%s\n",
						bsonId.StringValue())
					nc.conn.WriteBin(UPDATE_PREFIX, nodeId[:],
						uint16(len(newNode)), newNode)
					// utils.WriteBin(nc.conn, UPDATE_PREFIX,
					// nodeId[:], uint16(len(newNode)), newNode)
				}

			} else {
				log.Printf("nodeId=%x already in conns, for iddev=%x\n",
					nodeId[:], nc.iddev[:])
			}
		} else {
			if utils.ContainsWhere(&nodeId, nc.subs, comp) {
				nc.subs, _ = utils.Remove(nodeId, nc.subs)
				res := make(chan struct{})
				req := &subreq{res: res, nodeId: &nodeId, nc: nc}
				NCMS.GetMan(&nodeId).unsub <- req
				// NodeConnMan.unsub <- &subreq{nodeId: &nodeId, nc: nc}
				<-res
				close(res)
			} else {
				log.Printf("nodeId=%x not parts of conns for iddev=%x",
					nodeId[:], nc.iddev[:])
			}
		}
	}
}

func (m *man) Run() {
	sem := make(chan struct{}, 1000) // n routine lim
	var wg sync.WaitGroup
	log.Printf("man%d, running\n", m.i)
	for {
		select {
		case msg := <-m.BroadcastUpdate:
			log.Printf("man%d: broadcast request for nodeId=%x\n",
				m.i, msg.NodeId[:])
			// log.Println("Subs:", m.subs)
			// log.Println("Raw nodeId:", *msg.NodeId)

			if subs := m.subs[*msg.NodeId]; subs != nil {
				payload := msg.Payload
				// go func() {
				for _, x := range subs {
					wg.Add(1)
					sem <- struct{}{}
					go func() {
						defer wg.Done()
						defer func() { <-sem }()
						log.Printf("man%d: broadcasting to %x\n",
							m.i, x.iddev[:])
						x.conn.WriteBin(payload)
					}()
					// x.conn.Write(msg.Payload)
				}
				wg.Wait()
				//}()
			}

		case cl := <-m.clean:
			log.Printf("man%d: clean request for iddev=%x\n", m.i, cl.iddev[:])
			for _, x := range cl.subs {
				if node := m.subs[x]; m != nil {
					if u := node[*cl.iddev]; u != nil && u.sess == cl.sess {
						log.Printf("Removing from %x\n", x[:])
						delete(node, *cl.iddev)
						if len(node) == 0 {
							log.Printf("man%d: room for nodeId=%x"+
								"is empty, removing\n", m.i, x)
							delete(m.subs, x)
						}
					}
				}
			}
			// close(cl.res)
			cl.conn.C.Close()
			// cl.conn.Close()

		case sub := <-m.sub:
			log.Printf("man%d: subreq from iddev=%x to nodeId=%x\n",
				m.i, sub.nc.iddev[:], sub.nodeId[:])

			if subs_ := m.subs[*sub.nodeId]; subs_ == nil {
				log.Printf("man%d: creating sub group for nodeId=%x\n",
					m.i, sub.nodeId[:])
				subs_ = make(map[id_utils.Iddev_]*nconn)
				subs_[*sub.nc.iddev] = sub.nc
				m.subs[*sub.nodeId] = subs_
			} else if sub_ := subs_[*sub.nc.iddev]; sub_ != nil {
				log.Printf("man%d: already connected\n", m.i)
				if sub_.sess < sub.nc.sess {
					log.Printf("man%d: cur sess:%d new sess:%d\n",
						m.i, sub_.sess, sub.nc.sess)
					log.Printf("man%d: replacing with newer conn\n", m.i)
					// replace with new conn
					subs_[*sub.nc.iddev] = sub.nc
					// this will trigger clean up
					sub_.conn.C.Close()
				} else if sub_.sess == sub.nc.sess {
					log.Printf("man%d: req is from cur conn, ignoring\n",
						m.i)
				} else {
					log.Printf("man%d: req is from older conn, closing\n",
						m.i)
					// new req is older ... either rare or impossible
					// this will trigger cleanup
					sub.nc.conn.C.Close()
				}
			} else {
				log.Printf("man%d: adding the conn\n", m.i)
				subs_[*sub.nc.iddev] = sub.nc
			}
			sub.res <- struct{}{}
			// sub.nc.res <- struct{}{}

		case unsub := <-m.unsub:
			log.Printf("man%d: unsubreq from iddev=%x to nodeId=%x\n",
				m.i, unsub.nc.iddev[:], unsub.nodeId[:])
			if subs_ := m.subs[*unsub.nodeId]; subs_ != nil {
				if sub := subs_[*unsub.nc.iddev]; sub != nil {
					if sub.sess == unsub.nc.sess {
						log.Printf("man%d: unsubbing\n", m.i)
						delete(subs_, *unsub.nc.iddev)
						if len(subs_) == 0 {
							log.Printf("man%d: empty subs, rmving\n",
								m.i)
							delete(m.subs, *unsub.nodeId)
						}
					}
				}
			}
			unsub.res <- struct{}{}
			// unsub.nc.res <- struct{}{}
		}
	}
}

func handleNodeConn(conn net.Conn) {
	iddev := id_utils.Iddev_{}

	if _, err := conn.Read(iddev[:]); err != nil {
		log.Println("HandleNodeConn error reading iddev:", err)
		conn.Close()
		return
	}

	log.Printf("handleNodeConn: new node conn iddev=%x\n", iddev[:])

	nc := &nconn{
		sess:  utils.MakeTimestamp(),
		iddev: &iddev,
		conn:  utils.NewLockedConn(conn),
		subs:  make([]id_utils.NodeId, 0, 5),
		res:   make(chan error),
	}

	cm.addCh <- nc
	if err := <-nc.res; err != nil {
		log.Printf("handleNodeConn: err for %x, sess=%d: %v\n", *nc.iddev, nc.sess, err)
		close(nc.res)
		return
	}

	// no use after this, close right away
	close(nc.res)
	go nc.readFromConn()
}

// if err := utils.ReadBin(nc.conn.C, &connect, nodeId[:], &ts); err != nil {
// 	log.Printf("client=%x, error reading:\n%v\ndestroying\n",
// 		nc.iddev[:], err)
// 	NodeConnMan.clean <- nc
// 	return
// }

// if connect {
// 	// if not already connected, we fetch and write
// 	// the node if it has any update
// 	if !utils.ContainsWhere(&nodeId, nc.subs, comp) {
// 		nc.subs = append(nc.subs, nodeId)
// 		NodeConnMan.sub <- &subreq{nodeId: &nodeId, nc: nc}
// 		<-nc.res

// 		strId := hex.EncodeToString(nodeId[:])
// 		newNode, _ := GetMongoNodeByIdAfter(strId, ts)
// 		if newNode != nil {
// 			nc.conn.WriteBin(UPDATE_PREFIX, nodeId[:],
// 				uint16(len(newNode)), newNode)
// 			// utils.WriteBin(nc.conn, UPDATE_PREFIX, nodeId[:],
// 			// 	uint16(len(newNode)), newNode)
// 		}

// 	} else {
// 		log.Printf("nodeId=%x already in conns, for iddev=%x\n",
// 			nodeId[:], nc.iddev[:])
// 	}
// } else {
// 	if utils.ContainsWhere(&nodeId, nc.subs, comp) {
// 		nc.subs, _ = utils.Remove(nodeId, nc.subs)
// 		NodeConnMan.unsub <- &subreq{nodeId: &nodeId, nc: nc}
// 		<-nc.res
// 	} else {
// 		log.Printf("nodeId=%x not parts of conns for iddev=%x",
// 			nodeId[:], nc.iddev[:])
// 	}
// }

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
