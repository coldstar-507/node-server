package handlers

import (
	"encoding/hex"
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

type iddev = id_utils.Iddev_
type nodeid = id_utils.NodeId

type conns = map[iddev]*nconn3
type nodes = map[nodeid]*sconns

type sconns = utils.Smap[iddev, *nconn3]
type snodes = utils.Smap[nodeid, *sconns]

var n_man uint32
var n_sem uint32
var mans []*nodeMan

const DISC_CONF byte = 0x00
const CONN_CONF byte = 0x01

type nodeMan struct {
	i      uint32
	snodes *snodes
	sem    chan struct{}
}

func loadConfig() {
	const (
		nNodeKey = "N_NODE_MANAGERS"
		nSemKey  = "N_SEMAPHORE"
	)

	_nMan, err := strconv.Atoi(os.Getenv(nNodeKey))
	nMan := uint32(_nMan)
	utils.Panic(err, "loadConfig: ENV: undefined %s", nNodeKey)
	utils.Assert(nMan > 0, "loadConfig: %s needs to be a positive u32: %d", nNodeKey, nMan)
	n_man = nMan

	_nSem, err := strconv.Atoi(os.Getenv(nSemKey))
	nSem := uint32(_nSem)
	utils.Panic(err, "loadConfig: ENV: undefined %s", nSemKey)
	utils.Assert(nSem > 0, "loadConfig: %s needs to be a positive u32: %d", nSemKey, nSem)
	n_sem = nSem
}

func printMans() {
	for i, m := range mans {
		fmt.Printf("man %d:\n", i)
		m.snodes.Do(func(key nodeid, value *sconns) {
			fmt.Printf("\tnodeId %x\n", key)
			value.Dok(func(id iddev) {
				fmt.Printf("\t\tiddev %x\n", id)
			})
		})
	}
}

func StartNodeConnServer3() {
	listener, err := net.Listen("tcp", ":12000")
	utils.Panic(err, "StartNodeConnServer3: error on net.Listen")
	defer listener.Close()
	loadConfig()
	initNodeConnsManagers3()

	go func() {
		ticker := time.NewTicker(time.Second * 20)
		for range ticker.C {
			printMans()
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("error accepting connection:", err)
		} else {
			log.Println("new node connection:", conn.LocalAddr())
		}
		go handleNodeConn3(conn)
	}
}

func initNodeConnsManagers3() {
	mans = make([]*nodeMan, n_man)
	for i := range n_man {
		mans[i] = &nodeMan{
			i:      i,
			snodes: &snodes{M: make(nodes)},
			sem:    make(chan struct{}, n_sem),
		}
	}
}

func idf(nodeId nodeid) uint32 {
	h := fnv.New32()
	h.Write(nodeId[:])
	return h.Sum32() % n_man
}

func GetNodes(nodeId nodeid) *snodes {
	i := idf(nodeId)
	return mans[i].snodes
}

func GetMan(nodeId nodeid) *nodeMan {
	i := idf(nodeId)
	return mans[i]
}

func NodeBroadcast2(nodeId nodeid, payload []byte) {
	man := GetMan(nodeId)
	var nodecs []*nconn3
	man.snodes.ReadingAt(nodeId, func(e *sconns) {
		e.Reading(func(ncs conns) {
			nodecs = make([]*nconn3, 0, len(ncs))
			for _, nc := range ncs {
				nodecs = append(nodecs, nc)
			}
		})
	})

	wg := sync.WaitGroup{}
	for _, c := range nodecs {
		man.sem <- struct{}{}
		wg.Add(1)
		go func() {
			c.conn.WriteBin(payload)
			<-man.sem
			wg.Done()
		}()
	}
}

type nconn3 struct {
	// sess  int64
	iddev iddev
	conn  *utils.ClientConn
	// conn  net.Conn
	subs []nodeid
	// res  chan error
}

func handleNodeConn3(conn net.Conn) {
	iddev := iddev{}
	if _, err := conn.Read(iddev[:]); err != nil {
		log.Println("HandleNodeConn error reading iddev:", err)
		conn.Close()
		return
	}

	log.Printf("handleNodeConn: new node conn iddev=%x\n", iddev[:])

	nc := &nconn3{
		// sess:  utils.MakeTimestamp(),
		iddev: iddev,
		conn:  utils.NewLockedConn(conn),
		subs:  make([]nodeid, 0, 5),
	}

	go nc.readFromConn3()
}

func (nc *nconn3) readFromConn3() {
	log.Printf("client=%x listening for node connections\n", nc.iddev[:])
	defer log.Printf("client=%x done for node connections, closing thread\n", nc.iddev[:])

	var (
		err     error
		connect bool
		nodeId  = nodeid{}
		ts      int64
	)

	go func() {
		ticker := time.NewTicker(time.Second * 20)
		defer log.Printf("killed heartbeater for %x\n", nc.iddev)
		defer ticker.Stop()
		var heartbeatErr error
		for {
			<-ticker.C
			heartbeatErr = nc.conn.WriteBin(heartbeat)
			if heartbeatErr != nil {
				nc.conn.C.Close()
				return
			}
			log.Printf("client=%x, heartbeat", nc.iddev)
		}
	}()

	for {
		err = utils.ReadBin(nc.conn.C, &connect, nodeId[:], &ts)
		if err != nil {
			m := utils.SplitMap(nc.subs, idf)
			for i, subs := range m {
				man := mans[i]
				var md []nodeid
				man.snodes.Reading(func(e nodes) {
					for _, sub := range subs {
						v, ok := e[sub]
						if ok && v.Delete(nc.iddev) == 0 {
							md = append(md, sub)
						}
					}
				})

				if len(md) > 0 {
					man.snodes.Modifying(func(e nodes) {
						for _, m := range md {
							v, ok := e[m]
							if ok && len(v.M) == 0 {
								delete(e, m)
							}
						}
					})
				}

			}
			nc.conn.C.Close()
			return
		}

		if connect { // connecting to nodeId
			// if not already connected, we fetch and write
			// the node if it has any update
			connected := utils.Contains(nodeId, nc.subs)
			if !connected {
				man := GetNodes(nodeId)
				success := man.ReadingAt(nodeId, func(e *sconns) {
					if v, swapped := e.Swap(nc.iddev, nc); swapped {
						v.conn.C.Close()
					}
				})

				if !success {
					man.ModifyingAt(nodeId, func(s *sconns) {
						s.M[nc.iddev] = nc
					}, func() *sconns {
						sm := &sconns{M: make(conns)}
						sm.M[nc.iddev] = nc
						return sm
					})
				}

				nc.subs = append(nc.subs, nodeId)
				nc.conn.WriteBin(CONN_CONF, uint16(len(nodeId)), nodeId[:])

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
				}

			} else {
				nc.conn.WriteBin(CONN_CONF, uint16(len(nodeId)), nodeId[:])
				log.Printf("nodeId=%x already in conns, for iddev=%x\n",
					nodeId[:], nc.iddev[:])
			}
		} else { // disconnecting to nodeId
			if utils.Contains(nodeId, nc.subs) {
				var clearNode bool
				man := GetNodes(nodeId)
				man.ReadingAt(nodeId, func(e *sconns) {
					clearNode = e.Delete(nc.iddev) == 0
				})
				utils.Remove(nodeId, nc.subs)
				nc.conn.WriteBin(DISC_CONF, uint16(len(nodeId)), nodeId)
				if clearNode {
					man.DeleteIf(nodeId, func(value *sconns) bool {
						return len(value.M) == 0
					})
				}

			} else {
				nc.conn.WriteBin(DISC_CONF, uint16(len(nodeId)), nodeId)
				log.Printf("nodeId=%x not parts of conns for iddev=%x",
					nodeId[:], nc.iddev[:])
			}
		}
	}
}
