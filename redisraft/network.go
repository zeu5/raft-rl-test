package redisraft

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	lock       *sync.Mutex
	nodes      map[uint64]string
	eventTrace *EventTrace
	// Make this bag of messages
	messages   map[string]Message
	requests   map[string]int
	requestCtr int
}

func NewInterceptNetwork(ctx context.Context, addr string) *InterceptNetwork {

	f := &InterceptNetwork{
		Addr:       addr,
		ctx:        ctx,
		lock:       new(sync.Mutex),
		eventTrace: NewEventTrace(),
		nodes:      make(map[uint64]string),
		messages:   make(map[string]Message),
		requests:   make(map[string]int),
		requestCtr: 0,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.POST("/replica", f.handleReplica)
	r.POST("/event", f.handleEvent)
	r.POST("/message", f.handleMessage)
	f.server = &http.Server{
		Addr:    addr,
		Handler: r,
	}

	return f
}

func (n *InterceptNetwork) handleEvent(c *gin.Context) {
	event := make(map[string]interface{})
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}
	nodeID := 0
	nodeIDI, ok := event["replica"]
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

	eventTypeI, ok := event["type"]
	if !ok {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}
	eventType, ok := eventTypeI.(string)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}

	e := Event{
		Name:   eventType,
		Node:   nodeID,
		Params: n.mapEventToParams(eventType, event),
	}

	n.eventTrace.Add(e)
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (n *InterceptNetwork) getRequestNumber(req string) int {
	n.lock.Lock()
	defer n.lock.Unlock()

	ctr, ok := n.requests[req]
	if !ok {
		ctr = n.requestCtr
		n.requests[req] = ctr
		n.requestCtr += 1
	}

	return ctr
}

func (n *InterceptNetwork) mapEventToParams(eventType string, event map[string]interface{}) map[string]interface{} {
	params := make(map[string]interface{})
	eParams := event["params"].(map[string]interface{})
	switch eventType {
	case "ClientRequest":
		leader, _ := strconv.Atoi(eParams["leader"].(string))
		params["leader"] = leader
		params["request"] = n.getRequestNumber(eParams["request"].(string))
	case "BecomeLeader":
		node, _ := strconv.Atoi(eParams["node"].(string))
		term, _ := strconv.Atoi(eParams["term"].(string))
		params["node"] = node
		params["term"] = term
	case "Timeout":
		node, _ := strconv.Atoi(eParams["node"].(string))
		params["node"] = node
	case "MembershipChange":
		nodeI, ok := eParams["node"]
		if !ok || nodeI == nil {
			return params
		}
		node, _ := strconv.Atoi(nodeI.(string))
		actionI, ok := eParams["action"]
		if !ok || actionI == nil {
			return params
		}
		params["action"] = actionI.(string)
		params["node"] = node
	case "UpdateSnapshot":
		node, _ := strconv.Atoi(eParams["node"].(string))
		params["node"] = node
		params["snapshot_index"] = int(eParams["snapshot_index"].(float64))
	default:
		params = eParams
	}

	return params
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
	receiveEvent := Event{
		Name:   "SendMessage",
		Node:   int(m.From()),
		Params: n.getMessageEventParams(m),
	}
	n.eventTrace.Add(receiveEvent)

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

type entry struct {
	Term int    `json:"Term"`
	Data string `json:"Data"`
}

func (n *InterceptNetwork) getMessageEventParams(m Message) map[string]interface{} {
	params := make(map[string]interface{})

	params["term"] = int(m.ParsedMessage["term"].(float64))
	params["from"] = int(m.From())
	params["to"] = int(m.To())

	switch m.Type {
	case "append_entries_request":
		params["type"] = "MsgApp"
		params["log_term"] = m.ParsedMessage["prev_log_term"]
		entries := make([]entry, 0)
		for _, eI := range m.ParsedMessage["entries"].([]interface{}) {
			e := eI.(map[string]interface{})
			data := e["data"].(string)
			if data == "" {
				continue
			}
			eTermI, ok := e["term"]
			if !ok {
				continue
			}
			entries = append(entries, entry{
				Term: int(eTermI.(float64)),
				Data: strconv.Itoa(n.getRequestNumber(data)),
			})
		}
		params["entries"] = entries
		params["index"] = m.ParsedMessage["prev_log_idx"]
		params["commit"] = m.ParsedMessage["leader_commit"]
		params["reject"] = false
	case "append_entries_response":
		params["type"] = "MsgAppResp"
		params["log_term"] = 0
		params["entries"] = []entry{}
		params["index"] = m.ParsedMessage["current_idx"]
		params["commit"] = 0
		params["reject"] = int(m.ParsedMessage["success"].(float64)) == 0
	case "request_vote_request":
		params["type"] = "MsgVote"
		params["log_term"] = m.ParsedMessage["last_log_term"]
		params["entries"] = []entry{}
		params["index"] = m.ParsedMessage["last_log_idx"]
		params["commit"] = 0
		params["reject"] = false
	case "request_vote_response":
		params["type"] = "MsgVoteResp"
		params["log_term"] = 0
		params["entries"] = []entry{}
		params["index"] = 0
		params["commit"] = 0
		params["reject"] = int(m.ParsedMessage["vote_granted"].(float64)) == 0
	}
	return params
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
func (n *InterceptNetwork) Reset(epCtx *types.EpisodeContext) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.messages = make(map[string]Message)
	n.nodes = make(map[uint64]string)
	if epCtx.ShouldRecordEventTrace() {
		epCtx.RecordEventTrace(n.eventTrace)
	}

	n.eventTrace.Reset()
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
		// Transport: &http.Transport{
		// 	DialContext: (&net.Dialer{
		// 		Timeout:   5 * time.Second,
		// 		KeepAlive: 5 * time.Second,
		// 	}).DialContext,
		// 	TLSHandshakeTimeout:   5 * time.Second,
		// 	ResponseHeaderTimeout: 5 * time.Second,
		// 	ExpectContinueTimeout: 1 * time.Second,
		// 	DisableKeepAlives:     true,
		// },
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
	receiveEvent := Event{
		Name:   "DeliverMessage",
		Node:   int(m.To()),
		Params: n.getMessageEventParams(m),
	}
	n.eventTrace.Add(receiveEvent)
	// take the lock and delete the sent message from the list
	n.lock.Lock()
	delete(n.messages, id)
	n.lock.Unlock()

	return nil
}

func (n *InterceptNetwork) AddEvent(e Event) {
	n.eventTrace.Add(e)
}

// delete a message from the list given its id, if there is no such message => no-op
func (n *InterceptNetwork) DeleteMessage(id string) {
	n.lock.Lock()
	defer n.lock.Unlock()

	delete(n.messages, id)
}
