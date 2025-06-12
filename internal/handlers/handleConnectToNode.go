package handlers

// import (
// 	"bytes"
// 	"encoding/binary"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net/http"
// 	"time"
// )

// type longconn struct {
// 	iddev string
// 	id    string
// 	close chan struct{}
// 	w     io.Writer
// }

// type ConnMsg struct {
// 	Payload []byte
// 	Id      string
// }

// type initMsg struct {
// 	element   []byte
// 	id, iddev string
// }

// type connManager struct {
// 	name       string
// 	ticker     *time.Ticker
// 	addConn    chan *longconn
// 	initMsgCh  chan *initMsg
// 	conns      map[string](map[string]*longconn)
// 	getElement func(string) ([]byte, error)

// 	Broadcast chan *ConnMsg
// }

// // var UserConnectionManager = &connManager{
// // 	name:       "User",
// // 	ticker:     time.NewTicker(time.Second * 30),
// // 	addConn:    make(chan *longconn),
// // 	initMsgCh:  make(chan *initMsg),
// // 	conns:      make(map[string](map[string]*longconn)),
// // 	getElement: GetMongoUserById,

// // 	Broadcast: make(chan *ConnMsg),
// // }

// var NodeConnectionManager = &connManager{
// 	name:       "Node",
// 	ticker:     time.NewTicker(time.Second * 30),
// 	addConn:    make(chan *longconn),
// 	initMsgCh:  make(chan *initMsg),
// 	conns:      make(map[string](map[string]*longconn)),
// 	getElement: GetMongoNodeById,

// 	Broadcast: make(chan *ConnMsg),
// }

// func (cm *connManager) handleConn(id, iddev string, w io.Writer) (<-chan struct{}, error) {
// 	var conn *longconn
// 	if val, err := cm.getElement(id); err != nil {
// 		return nil, fmt.Errorf("%s handleConn error getting element: %v", cm.name, err)
// 	} else {
// 		conn = &longconn{iddev: iddev, id: id, w: w, close: make(chan struct{})}
// 		cm.addConn <- conn
// 		cm.initMsgCh <- &initMsg{element: val, id: id, iddev: iddev}
// 	}
// 	return conn.close, nil
// }

// func (cm *connManager) Run() {
// 	var (
// 		initMsgBuf   = bytes.NewBuffer(make([]byte, 0, 256))
// 		heartBeat    = []byte{0x99}
// 		toRemove     = make([]string, 0, 100)
// 		toRemoveRoom = make([]string, 0, 10)
// 	)
// 	for {
// 		select {
// 		case im := <-cm.initMsgCh:
// 			initMsgBuf.WriteByte(0x00)
// 			binary.Write(initMsgBuf, binary.BigEndian, uint16(len(im.element)))
// 			initMsgBuf.Write(im.element)
// 			if idconns := cm.conns[im.id]; idconns != nil {
// 				if td := idconns[im.iddev]; td != nil {
// 					log.Printf("%s init msg len=%d for id=%s, iddev=%s\n",
// 						cm.name, len(im.element), im.id, im.iddev)
// 					td.w.Write(initMsgBuf.Bytes())
// 					td.w.(http.Flusher).Flush()
// 				}
// 			}
// 			initMsgBuf.Reset()

// 		case <-cm.ticker.C:
// 			for nodeId, rooms := range cm.conns {
// 				for _, conn := range rooms {
// 					if _, err := conn.w.Write(heartBeat); err != nil {
// 						toRemove = append(toRemove, conn.iddev)
// 					} else {
// 						conn.w.(http.Flusher).Flush()
// 					}
// 				}

// 				for _, x := range toRemove {
// 					rooms[x].close <- struct{}{}
// 					close(rooms[x].close)
// 					delete(rooms, x)
// 					if len(rooms) == 0 {
// 						toRemoveRoom = append(toRemoveRoom, nodeId)
// 					}
// 				}
// 				toRemove = toRemove[:0]
// 			}

// 			for _, x := range toRemoveRoom {
// 				delete(cm.conns, x)
// 			}
// 			toRemoveRoom = toRemoveRoom[:0]

// 		case c := <-cm.addConn:
// 			if cm.conns[c.id] == nil {
// 				log.Printf("%s conns map id=%s is null, creating the map",
// 					cm.name, c.id)
// 				cm.conns[c.id] = make(map[string]*longconn)
// 			}
// 			if cc := cm.conns[c.id][c.iddev]; cc != nil {
// 				log.Printf("%s duplicate conn id=%s, iddev=%s: replacing it\n",
// 					cm.name, c.id, c.iddev)
// 				cc.close <- struct{}{}
// 				close(cc.close)
// 				delete(cm.conns[c.id], cc.iddev)
// 			}
// 			log.Printf("%s adding conn for id=%s, iddev=%s\n",
// 				cm.name, c.id, c.iddev)
// 			cm.conns[c.id][c.iddev] = c

// 		case m := <-cm.Broadcast:
// 			log.Printf("%s broadcast for id=%s\n", cm.name, m.Id)
// 			if cm.conns[m.Id] == nil {
// 				log.Printf("%s, no one listening for changes for id=%s\n",
// 					cm.name, m.Id)
// 			} else {
// 				for _, conn := range cm.conns[m.Id] {
// 					log.Printf("%s broadcasting for id=%s to iddev=%s\n",
// 						cm.name, m.Id, conn.iddev)
// 					conn.w.Write(m.Payload)
// 					conn.w.(http.Flusher).Flush()
// 				}
// 			}
// 		}
// 	}

// }

// func HandleNodeConnection(w http.ResponseWriter, r *http.Request) {
// 	defer r.Body.Close()
// 	id, iddev := r.PathValue("id"), r.PathValue("iddev")
// 	close, err := NodeConnectionManager.handleConn(id, iddev, w)
// 	if err != nil {
// 		w.WriteHeader(500)
// 		log.Println("HandleUserConnection error:", err)
// 		return
// 	}
// 	<-close
// 	log.Printf("Terminating Node conn for id=%s, iddev=%s\n", id, iddev)
// }

// func HandleUserConnection(w http.ResponseWriter, r *http.Request) {
// 	defer r.Body.Close()
// 	id, iddev := r.PathValue("id"), r.PathValue("iddev")
// 	close, err := UserConnectionManager.handleConn(id, iddev, w)
// 	if err != nil {
// 		w.WriteHeader(500)
// 		log.Println("HandleUserConnection error:", err)
// 		return
// 	}
// 	<-close
// 	log.Printf("Terminating User conn for id=%s, iddev=%s\n", id, iddev)
// }
