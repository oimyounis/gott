package gott

import (
	gob "bytes"
	"fmt"
	"gott/utils"
	"log"
	"strings"
)

var indent = "  "

type Subscription struct {
	Client *Client
	QoS    byte
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
		if gob.Equal(b, TOPIC_SINGLE_LEVEL_WILDCARD) {
			tl.hasSingleWildcardAsChild = true
		} else if gob.Equal(b, TOPIC_MULTI_LEVEL_WILDCARD) {
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
		if tl.hasSingleWildcardAsChild && !singleWildcardFound && gob.Equal(f.bytes, TOPIC_SINGLE_LEVEL_WILDCARD) {
			matches = append(matches, f)
			singleWildcardFound = true
		}

		if tl.hasMultiWildcardAsChild && !multiWildcardFound && gob.Equal(f.bytes, TOPIC_MULTI_LEVEL_WILDCARD) {
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
		if sub.Client.ClientId == client.ClientId {
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
		strs = append(strs, fmt.Sprintf("%v:%v", s.Client.ClientId, s.QoS))
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
	if gob.Equal(tl.bytes, TOPIC_MULTI_LEVEL_WILDCARD) {
		ts.hasGlobalFilter = true
	} else if gob.Equal(tl.bytes, TOPIC_SINGLE_LEVEL_WILDCARD) {
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
	globalFilterFound := false
	topSingleWildcardFound := false

	for _, f := range ts.filters {
		if !targetFound && gob.Equal(f.bytes, b) {
			matches = append(matches, f)
			targetFound = true
		}
		if ts.hasGlobalFilter && !globalFilterFound && gob.Equal(f.bytes, TOPIC_MULTI_LEVEL_WILDCARD) {
			matches = append(matches, f)
			globalFilterFound = true
		}
		if ts.hasTopSingleWildcard && !topSingleWildcardFound && gob.Equal(f.bytes, TOPIC_SINGLE_LEVEL_WILDCARD) {
			matches = append(matches, f)
			topSingleWildcardFound = true
		}

		if targetFound && (!ts.hasGlobalFilter || globalFilterFound) && (!ts.hasTopSingleWildcard || topSingleWildcardFound) {
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

func (ts *TopicStorage) Match(topic []byte) []*TopicLevel {
	matches := make([]*TopicLevel, 0)

	segs := gob.Split(topic, TOPIC_DELIM)

	hits := ts.FindAll(segs[0])

	log.Printf("hits: %s", hits)

	for _, hit := range hits {
		match(hit, segs[1:], &matches)
	}

	return matches
}

func match(t *TopicLevel, segs [][]byte, matches *[]*TopicLevel) *TopicLevel {
	if (len(t.Subscriptions) != 0 || len(t.children) == 0) && len(segs) == 0 || (gob.Equal(t.bytes, TOPIC_SINGLE_LEVEL_WILDCARD) && len(t.children) == 0 && len(segs) == 0) || gob.Equal(t.bytes, TOPIC_MULTI_LEVEL_WILDCARD) {
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

func ValidFilter(filter []byte) bool {
	multiWildcard := gob.IndexByte(filter, TOPIC_MULTI_LEVEL_WILDCARD[0])
	singleWildcards := utils.IndexAllByte(filter, TOPIC_SINGLE_LEVEL_WILDCARD[0])
	filterLen := len(filter)

	if filterLen == 0 {
		return false
	}

	if gob.Count(filter, TOPIC_MULTI_LEVEL_WILDCARD) > 1 || multiWildcard != -1 && multiWildcard != filterLen-1 || (multiWildcard > 0 && filter[multiWildcard-1] != TOPIC_DELIM[0]) {
		return false
	}

	for _, idx := range singleWildcards {
		if idx > 0 && filter[idx-1] != TOPIC_DELIM[0] {
			return false
		} else if filterLen > 1 && idx == 0 && idx != filterLen-1 && filter[idx+1] != TOPIC_DELIM[0] {
			return false
		}
	}

	return true
}
