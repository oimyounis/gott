package main

import (
	gob "bytes"
	"fmt"
	"gott"
	"log"
	"strings"
	"time"
)

var TOPIC_FILTER1 = []byte("things/device-81752/+/data")
var TOPIC_FILTER1_2 = []byte("things/device-81752/+/cmd")
var TOPIC_FILTER1_3 = []byte("things/device-81752/+/cmd/#")

var TOPIC_FILTER2 = []byte("things/device-11752/+/data")
var TOPIC_FILTER2_1 = []byte("things/device-11752/+/")

var TOPIC_FILTER3 = []byte("/")

var TOPIC_FILTER4 = []byte("#")

var TOPIC_FILTER5 = []byte("test")

var TOPIC_FILTER6 = []byte("/test")

var TOPIC_FILTER7 = []byte("test/")

var TOPIC_FILTER8 = []byte("test/data")
var TOPIC_FILTER9 = []byte("test/data/+/+")

var TOPIC_FILTER10 = []byte("+")

var TOPIC_FILTER11 = []byte("test/data/coords")
var TOPIC_FILTER12 = []byte("test/data/+")

var PUB_TOPIC = []byte{49, 116, 104, 105, 110, 103, 115, 47, 100, 101, 118, 105, 99, 101, 45, 52, 56, 48, 55, 55, 47, 100, 97, 116, 97, 47, 112, 111, 115, 105, 116, 105, 111, 110, 0, 1, 116, 101, 115, 116}

var indent = "  "

type Subscription struct {
	Client *Client
	QoS    byte
}

type Client struct {
	Name string
}

func (c *Client) emit(topic, msg, qos []byte) {
	log.Println("emitting <", c.Name, string(topic), string(msg), qos)
}

func (c *Client) String() string {
	return c.Name
}

type Message struct {
	Payload []byte
	QoS     byte
}

type TopicLevel struct {
	bytes                    []byte
	children                 []*TopicLevel
	hasSingleWildcardAsChild bool
	hasMultiWildcardAsChild  bool
	parent                   *TopicLevel
	Subscriptions            []*Subscription
	QoS                      byte
	RetainedMessage          *Message
}

//func (tl *TopicLevel) MatchSub(children [][]byte) []*Subscription {
//
//}

func (tl *TopicLevel) AddChild(child *TopicLevel) {
	tl.children = append(tl.children, child)
}

func (tl *TopicLevel) ParseChildren(client *Client, children [][]byte, qos byte) {
	childrenLen := len(children)
	if childrenLen == 0 {
		return
	}

	b := children[0]
	l := tl.Find(b)
	if l == nil {
		l = &TopicLevel{bytes: b, parent: tl}
		tl.AddChild(l)
		if gob.Equal(b, gott.TOPIC_SINGLE_LEVEL_WILDCARD) {
			tl.hasSingleWildcardAsChild = true
		} else if gob.Equal(b, gott.TOPIC_MULTI_LEVEL_WILDCARD) {
			tl.hasMultiWildcardAsChild = true
			tl.CreateOrUpdateSubscription(client, qos)
		}
	}

	if childrenLen == 1 {
		l.CreateOrUpdateSubscription(client, qos)
		return
	}
	l.ParseChildren(client, children[1:], qos)
}

func (tl *TopicLevel) Find(child []byte) *TopicLevel {
	for _, c := range tl.children {
		if gob.Equal(c.bytes, child) {
			return c
		}
	}
	return nil
}

func (tl *TopicLevel) FindAll(b []byte) (matches []*TopicLevel) {
	targetFound := false
	singleWildcardFound := false
	multiWildcardFound := false

	for _, f := range tl.children {
		if !targetFound && gob.Equal(f.bytes, b) {
			matches = append(matches, f)
			targetFound = true
		}
		if tl.hasSingleWildcardAsChild && !singleWildcardFound && gob.Equal(f.bytes, gott.TOPIC_SINGLE_LEVEL_WILDCARD) {
			matches = append(matches, f)
			singleWildcardFound = true
		}

		if tl.hasMultiWildcardAsChild && !multiWildcardFound && gob.Equal(f.bytes, gott.TOPIC_MULTI_LEVEL_WILDCARD) {
			matches = append(matches, f)
			multiWildcardFound = true
		}

		if targetFound && (!tl.hasSingleWildcardAsChild || singleWildcardFound) && (!tl.hasMultiWildcardAsChild || multiWildcardFound) {
			break
		}
	}
	return
}

func (tl *TopicLevel) CreateOrUpdateSubscription(client *Client, qos byte) {
	for _, sub := range tl.Subscriptions {
		if sub.Client.Name == client.Name {
			sub.QoS = qos
			return
		}
	}
	tl.Subscriptions = append(tl.Subscriptions, &Subscription{
		Client: client,
		QoS:    qos,
	})
}

func (tl *TopicLevel) Print(add string) {
	log.Println(add, tl.String(), "- subscriptions:", tl.SubscriptionsString())
	for _, c := range tl.children {
		c.Print(add + indent)
	}
}

func (tl *TopicLevel) SubscriptionsString() string {
	strs := make([]string, 0)
	for _, s := range tl.Subscriptions {
		strs = append(strs, fmt.Sprintf("%v:%v", s.Client.Name, s.QoS))
	}

	return strings.Join(strs, ", ")
}

func (tl *TopicLevel) Name() string {
	return string(tl.bytes)
}

func (tl *TopicLevel) Path() string {
	if tl.parent != nil {
		return tl.parent.Path() + "/" + tl.Name()
	}
	return tl.Name()
}

func (tl *TopicLevel) String() string {
	return tl.Path()
}

type TopicStorage struct {
	filters              []*TopicLevel
	hasGlobalFilter      bool // has "#"
	hasTopSingleWildcard bool // has "+" as top level
}

func (ts *TopicStorage) AddTopLevel(tl *TopicLevel) {
	ts.filters = append(ts.filters, tl)
	if gob.Equal(tl.bytes, gott.TOPIC_MULTI_LEVEL_WILDCARD) {
		ts.hasGlobalFilter = true
	} else if gob.Equal(tl.bytes, gott.TOPIC_SINGLE_LEVEL_WILDCARD) {
		ts.hasTopSingleWildcard = true
	}
}

func (ts *TopicStorage) Find(b []byte) *TopicLevel {
	for _, f := range ts.filters {
		if gob.Equal(f.bytes, b) {
			return f
		}
	}
	return nil
}

func (ts *TopicStorage) FindAll(b []byte) (matches []*TopicLevel) {
	targetFound := false
	globalFound := false
	topSingleWildcardFound := false

	for _, f := range ts.filters {
		if !targetFound && gob.Equal(f.bytes, b) {
			matches = append(matches, f)
			targetFound = true
		}
		if ts.hasGlobalFilter && !globalFound && gob.Equal(f.bytes, gott.TOPIC_MULTI_LEVEL_WILDCARD) {
			matches = append(matches, f)
			globalFound = true
		}
		if ts.hasTopSingleWildcard && !topSingleWildcardFound && gob.Equal(f.bytes, gott.TOPIC_SINGLE_LEVEL_WILDCARD) {
			matches = append(matches, f)
			topSingleWildcardFound = true
		}

		if targetFound && (!ts.hasGlobalFilter || globalFound) && (!ts.hasTopSingleWildcard || topSingleWildcardFound) {
			break
		}
	}
	return
}

func (ts *TopicStorage) Print() {
	for _, f := range ts.filters {
		f.Print(indent)
	}
}

func match(t *TopicLevel, segs [][]byte, matches *[]*TopicLevel) *TopicLevel {
	if (len(t.Subscriptions) != 0 || len(t.children) == 0) && len(segs) == 0 || (gob.Equal(t.bytes, gott.TOPIC_SINGLE_LEVEL_WILDCARD) && len(t.children) == 0 && len(segs) == 0) || gob.Equal(t.bytes, gott.TOPIC_MULTI_LEVEL_WILDCARD) {
		*matches = append(*matches, t)
		return t
	}

	if len(t.children) != 0 && len(segs) != 0 {
		if t.hasMultiWildcardAsChild && len(segs) == 0 {
			*matches = append(*matches, t)
		}
		hits := t.FindAll(segs[0])
		for _, hit := range hits {
			match(hit, segs[1:], matches)
		}
	}

	return nil
}

func (ts *TopicStorage) Match(topic []byte) []*TopicLevel {
	matches := make([]*TopicLevel, 0)

	segs := gob.Split(topic, gott.TOPIC_DELIM)

	hits := storage.FindAll(segs[0])

	log.Printf("hits: %s", hits)

	for _, hit := range hits {
		match(hit, segs[1:], &matches)
	}

	return matches
}

var storage = &TopicStorage{}

func subscribe(client *Client, bytes []byte, qos byte) {
	segs := gob.Split(bytes, gott.TOPIC_DELIM)
	//log.Println("segs", segs)

	segsLen := len(segs)
	if segsLen == 0 {
		return
	}

	topLevel := segs[0]
	tl := storage.Find(topLevel)
	if tl == nil {
		tl = &TopicLevel{bytes: topLevel}
		storage.AddTopLevel(tl)
	}
	if segsLen == 1 {
		tl.CreateOrUpdateSubscription(client, qos)
		return
	}
	tl.ParseChildren(client, segs[1:], qos)

}

func publish(topic, msg []byte, qos byte) {
	// NOTE: the server never upgrades QoS levels, downgrades only when necessary as in Min(pub.QoS, sub.QoS)
	m := storage.Match(topic)
	log.Println(string(topic), "matches", m)
}

func main() {
	//client1 := &Client{Name: "client1"}
	//client2 := &Client{Name: "client2"}
	//client3 := &Client{Name: "client3"}

	now := time.Now()
	clients := make([]*Client, 0)
	i := 0
	for len(clients) < 50 {
		i++
		clients = append(clients, &Client{Name: fmt.Sprintf("client%d", i)})
	}

	subscribe(clients[0], TOPIC_FILTER1, 0)
	subscribe(clients[2], TOPIC_FILTER1, 1)
	subscribe(clients[0], TOPIC_FILTER2, 1)
	subscribe(clients[1], TOPIC_FILTER2_1, 2)
	subscribe(clients[2], TOPIC_FILTER2_1, 1)
	subscribe(clients[11], TOPIC_FILTER1_2, 1)
	subscribe(clients[10], TOPIC_FILTER3, 0)
	//
	subscribe(clients[30], TOPIC_FILTER4, 0)
	subscribe(clients[19], TOPIC_FILTER4, 0)
	//
	subscribe(clients[1], TOPIC_FILTER1_3, 1)
	subscribe(clients[0], TOPIC_FILTER5, 1)
	subscribe(clients[22], TOPIC_FILTER6, 1)
	subscribe(clients[13], TOPIC_FILTER6, 2)
	subscribe(clients[21], TOPIC_FILTER7, 2)
	//
	subscribe(clients[20], TOPIC_FILTER8, 0)
	subscribe(clients[28], TOPIC_FILTER8, 2)
	subscribe(clients[40], TOPIC_FILTER8, 0)
	subscribe(clients[45], TOPIC_FILTER1, 0)
	subscribe(clients[6], TOPIC_FILTER1, 1)
	subscribe(clients[39], TOPIC_FILTER2, 1)
	subscribe(clients[32], TOPIC_FILTER2_1, 2)
	subscribe(clients[24], TOPIC_FILTER2_1, 1)
	subscribe(clients[17], TOPIC_FILTER1_2, 1)
	subscribe(clients[16], TOPIC_FILTER3, 0)
	//
	subscribe(clients[8], TOPIC_FILTER4, 0)
	subscribe(clients[9], TOPIC_FILTER4, 0)
	//
	subscribe(clients[14], TOPIC_FILTER1_3, 1)
	subscribe(clients[36], TOPIC_FILTER5, 1)
	subscribe(clients[48], TOPIC_FILTER6, 1)
	subscribe(clients[44], TOPIC_FILTER6, 2)
	subscribe(clients[33], TOPIC_FILTER7, 2)
	//
	subscribe(clients[11], TOPIC_FILTER8, 0)
	subscribe(clients[4], TOPIC_FILTER8, 2)
	subscribe(clients[7], TOPIC_FILTER8, 0)
	subscribe(clients[16], TOPIC_FILTER1, 0)
	subscribe(clients[24], TOPIC_FILTER1, 1)
	subscribe(clients[43], TOPIC_FILTER2, 1)
	subscribe(clients[41], TOPIC_FILTER2_1, 2)
	subscribe(clients[32], TOPIC_FILTER2_1, 1)
	subscribe(clients[41], TOPIC_FILTER1_2, 1)
	subscribe(clients[36], TOPIC_FILTER3, 0)
	//
	subscribe(clients[0], TOPIC_FILTER4, 0)
	subscribe(clients[1], TOPIC_FILTER4, 0)
	//
	subscribe(clients[40], TOPIC_FILTER1_3, 1)
	subscribe(clients[32], TOPIC_FILTER5, 1)
	subscribe(clients[25], TOPIC_FILTER6, 1)
	subscribe(clients[1], TOPIC_FILTER6, 2)
	subscribe(clients[2], TOPIC_FILTER7, 2)
	//
	subscribe(clients[1], TOPIC_FILTER8, 0)
	subscribe(clients[10], TOPIC_FILTER8, 2)
	subscribe(clients[0], TOPIC_FILTER8, 0)
	//
	//subscribe(client2, TOPIC_FILTER9, 1)
	//subscribe(client1, TOPIC_FILTER9, 1)
	//subscribe(client3, TOPIC_FILTER9, 0)

	//subscribe(client1, TOPIC_FILTER10, 1)
	//subscribe(client3, TOPIC_FILTER12, 0)
	//subscribe(client2, TOPIC_FILTER11, 0)
	//
	//storage.Print()

	//top := []byte("test/dada")

	//publish(TOPIC_FILTER11, []byte("test123"), 0)
	//publish([]byte("devices"), []byte("test123"), 0)

	//subscribe(clients[0], b("devices/a/b"), 0)
	//subscribe(clients[1], b("devices/a/b"), 0)
	//subscribe(clients[2], b("devices/a/c"), 0)
	//subscribe(clients[3], b("devices/a/c"), 0)
	//subscribe(clients[4], b("devices/b/a"), 0)
	//subscribe(clients[5], b("things/dev/remote/cmd"), 0)
	//subscribe(clients[6], b("/sports"), 0)
	//subscribe(clients[6], b("sports test"), 0)
	//subscribe(clients[6], b("sports/"), 0)
	//subscribe(clients[7], b("/"), 0)
	//subscribe(clients[7], b("sport/tennis/player1"), 0)
	//subscribe(clients[8], b("sport/tennis/player1/#"), 0)
	subscribe(clients[8], b("+"), 0)

	storage.Print()

	publish([]byte("things/device-11752/sensors/data"), b("test123"), 0)
	publish([]byte("things/device-11752/cmds/"), b("test123"), 0)
	//publish(TOPIC_FILTER7, b("test123"), 0)

	log.Println("took:", time.Since(now))

	//log.Println(bytes.BinaryStringToByte("0011"))
	//gott.MakePublishPacket([]byte("aa/b"), []byte(""), 1, 1, 1)
	//gott.MakePublishPacket([]byte("aa/b"), []byte(""), 1, 0, 1)
	//gott.MakePublishPacket([]byte("a/b"), []byte(""), 0, 2, 0)
	//broker := gott.NewBroker(":1883")
	//if err := broker.Listen(); err != nil {
	//	panic(err)
	//}
}

func b(str string) []byte {
	return []byte(str)
}
