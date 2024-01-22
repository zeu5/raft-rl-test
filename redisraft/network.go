package redisraft

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zeu5/raft-rl-test/types"
)

type Message struct {
	Fr            string                 `json:"from"`
	T             string                 `json:"to"`
	Data          string                 `json:"data"`
	Type          string                 `json:"type"`
	ID            string                 `json:"id"`
	ParsedMessage map[string]interface{} `json:"-"`
}

func (m Message) To() uint64 {
	to, _ := strconv.Atoi(m.T)
	return uint64(to)
}

func (m Message) From() uint64 {
	from, _ := strconv.Atoi(m.Fr)
	return uint64(from)
}

func (m Message) Copy() Message {
	n := Message{
		Fr:            m.Fr,
		T:             m.T,
		Data:          m.Data,
		Type:          m.Type,
		ID:            m.ID,
		ParsedMessage: make(map[string]interface{}),
	}
	if m.ParsedMessage != nil {
		for k, v := range m.ParsedMessage {
			n.ParsedMessage[k] = v
		}
	}
	return n
}

func (m Message) Hash() string {
	return m.ID
}

var _ types.Message = Message{}

type InterceptNetwork struct {
	Addr   string
	ctx    context.Context
	server *http.Server

	lock  *sync.Mutex
	nodes map[uint64]string
	// Make this bag of messages
	messages map[string]Message
}

func NewInterceptNetwork(ctx context.Context, addr string) *InterceptNetwork {

	f := &InterceptNetwork{
		Addr:     addr,
		ctx:      ctx,
		lock:     new(sync.Mutex),
		nodes:    make(map[uint64]string),
		messages: make(map[string]Message),
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.POST("/replica", f.handleReplica)
	r.POST("/event", dummyHandler)
	r.POST("/message", f.handleMessage)
	f.server = &http.Server{
		Addr:    addr,
		Handler: r,
	}

	return f
}

func dummyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// copies all the messages from the InterceptNetwork, should not affect the network or other episodes
func (n *InterceptNetwork) GetAllMessages() map[string]Message {
	out := make(map[string]Message)
	n.lock.Lock()
	defer n.lock.Unlock()
	for k, m := range n.messages {
		out[k] = m.Copy()
	}
	return out
}

func (n *InterceptNetwork) handleMessage(c *gin.Context) {
	m := Message{}
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}
	parsedMessage := make(map[string]interface{})
	if err := json.Unmarshal([]byte(m.Data), &parsedMessage); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}
	m.ParsedMessage = parsedMessage

	n.lock.Lock()
	n.messages[m.ID] = m
	n.lock.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (n *InterceptNetwork) handleReplica(c *gin.Context) {
	replica := make(map[string]interface{})
	if err := c.ShouldBindJSON(&replica); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}
	nodeID := 0
	nodeIDI, ok := replica["id"]
	if !ok {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}
	nodeIDS, ok := nodeIDI.(string)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}
	nodeID, err := strconv.Atoi(nodeIDS)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}

	nodeAddrI, ok := replica["addr"]
	if !ok {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}
	nodeAddr, ok := nodeAddrI.(string)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}

	n.lock.Lock()
	n.nodes[uint64(nodeID)] = nodeAddr
	n.lock.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (n *InterceptNetwork) Start() {
	go func() { // starts the server to listen for requests
		n.server.ListenAndServe()
	}()

	go func() { // what is this routine doing? wait for cancel signal and shutdown the server
		<-n.ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		n.server.Shutdown(ctx)
	}()
}

// re-create the messages and nodes maps for the network
func (n *InterceptNetwork) Reset() {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.messages = make(map[string]Message)
	n.nodes = make(map[uint64]string)
}

// wait until the specified number of nodes get connected, return false if it does not happen within the internal specified timeout
func (n *InterceptNetwork) WaitForNodes(numNodes int) bool {
	timeout := time.After(2 * time.Second)
	numConnectedNodes := 0
	for numConnectedNodes != numNodes {
		select {
		case <-n.ctx.Done():
			return false
		case <-timeout:
			return false
		case <-time.After(1 * time.Millisecond):
		}
		n.lock.Lock()
		numConnectedNodes = len(n.nodes)
		n.lock.Unlock()
	}
	return true
}
func (n *InterceptNetwork) SendMessage(id string, epCtx *types.EpisodeContext) error {
	var start time.Time

	start = time.Now()
	n.lock.Lock()
	m, ok1 := n.messages[id] // get message from the list
	nodeAddr := ""
	if ok1 { // if present, read the target node address
		nodeAddr = n.nodes[m.To()]
	}
	n.lock.Unlock()
	epCtx.Report.AddTimeEntry(time.Since(start), "net_send_msg_get_msg", "InterceptNetwork.SendMessageCtx")

	if !ok1 { // if message not present, return
		return errors.New("SendMessage : read message is invalid")
	}

	start = time.Now()
	// marshal the message to send it
	bs, err := json.Marshal(m)
	if err != nil {
		return errors.New("SendMessage : error marshaling the message")
	}
	epCtx.Report.AddTimeEntry(time.Since(start), "net_send_msg_marshal", "InterceptNetwork.SendMessageCtx")

	start = time.Now()
	// set up http client to send it?
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     true,
		},
	}
	epCtx.Report.AddTimeEntry(time.Since(start), "net_send_msg_setup_client", "InterceptNetwork.SendMessageCtx")

	start = time.Now()
	// send the message over http
	resp, err := client.Post("http://"+nodeAddr+"/message", "application/json", bytes.NewBuffer(bs))
	epCtx.Report.AddTimeEntry(time.Since(start), "net_send_msg_post", "InterceptNetwork.SendMessageCtx")
	if err == nil { // what happens here?
		start = time.Now()
		io.ReadAll(resp.Body)
		resp.Body.Close()
		epCtx.Report.AddTimeEntry(time.Since(start), "net_send_msg_post_read", "InterceptNetwork.SendMessageCtx")
	} else {
		return fmt.Errorf(fmt.Sprintf("SendMessage : error with post operation \n%s", err))
	}

	// take the lock and delete the sent message from the list
	n.lock.Lock()
	delete(n.messages, id)
	n.lock.Unlock()

	return nil
}

// delete a message from the list given its id, if there is no such message => no-op
func (n *InterceptNetwork) DeleteMessage(id string) {
	n.lock.Lock()
	defer n.lock.Unlock()

	delete(n.messages, id)
}
