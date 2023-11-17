package cbft

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zeu5/raft-rl-test/types"
)

type Message struct {
	FromAlias     string          `json:"from"`
	ToAlias       string          `json:"to"`
	Data          string          `json:"data"`
	Type          string          `json:"type"`
	ID            string          `json:"id"`
	ParsedMessage *WrappedEnvelop `json:"-"`
	FromID        int             `json:"-"`
	ToID          int             `json:"-"`
}

type WrappedEnvelop struct {
	ChanID byte
	Msg    []byte
}

func (m Message) To() uint64 {
	return uint64(m.ToID)
}

func (m Message) From() uint64 {
	return uint64(m.FromID)
}

func (m Message) Copy() Message {
	n := Message{
		FromAlias: m.FromAlias,
		ToAlias:   m.ToAlias,
		Data:      m.Data,
		Type:      m.Type,
		ID:        m.ID,
		FromID:    m.FromID,
		ToID:      m.ToID,
	}
	if m.ParsedMessage != nil {
		n.ParsedMessage = &WrappedEnvelop{
			ChanID: m.ParsedMessage.ChanID,
			Msg:    make([]byte, len(m.ParsedMessage.Msg)),
		}
		copy(n.ParsedMessage.Msg, m.ParsedMessage.Msg)
	}
	return n
}

func (m Message) Hash() string {
	return m.ID
}

var _ types.Message = Message{}

type nodeKeys struct {
	Public  []byte
	Private []byte
}
type nodeInfo struct {
	ID    int
	Alias string
	Addr  string
	Keys  *nodeKeys
}

type InterceptNetwork struct {
	Port   int
	ctx    context.Context
	server *http.Server

	lock     *sync.Mutex
	nodes    map[int]*nodeInfo
	aliasMap map[string]int
	// Make this bag of messages
	messages map[string]Message
}

func NewInterceptNetwork(ctx context.Context, port int) *InterceptNetwork {

	f := &InterceptNetwork{
		Port:     port,
		ctx:      ctx,
		lock:     new(sync.Mutex),
		nodes:    make(map[int]*nodeInfo),
		aliasMap: make(map[string]int),
		messages: make(map[string]Message),
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.POST("/replica", f.handleReplica)
	r.POST("/event", dummyHandler)
	r.POST("/message", f.handleMessage)
	f.server = &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: r,
	}

	return f
}

func dummyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

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
	data, err := base64.StdEncoding.DecodeString(m.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}
	fmt.Println("Received message of type: " + m.Type)
	we := &WrappedEnvelop{}
	if err := json.Unmarshal(data, we); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}
	m.ParsedMessage = we

	n.lock.Lock()
	m.FromID = n.aliasMap[m.FromAlias]
	m.ToID = n.aliasMap[m.ToAlias]
	n.messages[m.ID] = m
	n.lock.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (n *InterceptNetwork) handleReplica(c *gin.Context) {
	nodeI := &nodeInfo{}
	if err := c.ShouldBindJSON(&nodeI); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}

	n.lock.Lock()
	n.nodes[nodeI.ID] = nodeI
	n.aliasMap[nodeI.Alias] = nodeI.ID
	n.lock.Unlock()

	fmt.Println("Received replica")

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (n *InterceptNetwork) Start() {
	go func() {
		n.server.ListenAndServe()
	}()

	go func() {
		<-n.ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		n.server.Shutdown(ctx)
	}()
}

func (n *InterceptNetwork) Reset() {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.messages = make(map[string]Message)
	n.nodes = make(map[int]*nodeInfo)
	n.aliasMap = make(map[string]int)
}

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

func (n *InterceptNetwork) SendMessage(id string) {
	n.lock.Lock()
	m, ok := n.messages[id]
	nodeAddr := ""
	if ok {
		nodeAddr = n.nodes[m.ToID].Addr
	}
	n.lock.Unlock()

	if !ok {
		return
	}

	bs, err := json.Marshal(m)
	if err != nil {
		return
	}
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
	resp, err := client.Post("http://"+nodeAddr+"/message", "application/json", bytes.NewBuffer(bs))
	if err == nil {
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}

	n.lock.Lock()
	delete(n.messages, id)
	n.lock.Unlock()
}

func (n *InterceptNetwork) DeleteMessage(id string) {
	n.lock.Lock()
	defer n.lock.Unlock()

	delete(n.messages, id)
}
