package gott

import (
	gob "bytes"
	"fmt"
	"gott/utils"
	"strings"
)

var indent = "  "

type TopicLevel struct {
	hasSingleWildcardAsChild bool
	hasMultiWildcardAsChild  bool
	parent                   *TopicLevel
	Bytes                    []byte
	Children                 []*TopicLevel
	Subscriptions            []*Subscription
	QoS                      byte
	RetainedMessage          *Message
}

func (tl *TopicLevel) deleteSubscription(index int) {
	var newSubs []*Subscription
	newSubs = append(newSubs, tl.Subscriptions[:index]...)
	if index != len(tl.Subscriptions)-1 {
		newSubs = append(newSubs, tl.Subscriptions[index+1:]...)
	}
	tl.Subscriptions = newSubs
}

func (tl *TopicLevel) reverse(segs [][]byte, matches *[]*TopicLevel) {
	if len(segs) == 0 {
		if !gob.Equal(tl.Bytes, TOPIC_MULTI_LEVEL_WILDCARD) && tl.RetainedMessage != nil {
			*matches = append(*matches, tl)
		}
		return
	}

	seg := segs[0]

	isSingleWildcard := gob.Equal(seg, TOPIC_SINGLE_LEVEL_WILDCARD)
	isMultiWildcard := gob.Equal(seg, TOPIC_MULTI_LEVEL_WILDCARD)

	if isMultiWildcard && !gob.Equal(tl.Bytes, TOPIC_MULTI_LEVEL_WILDCARD) && tl.RetainedMessage != nil {
		*matches = append(*matches, tl)
	}

	for _, child := range tl.Children {
		if isMultiWildcard {
			child.reverse([][]byte{TOPIC_MULTI_LEVEL_WILDCARD}, matches)
		} else if isSingleWildcard || gob.Equal(seg, child.Bytes) {
			child.reverse(segs[1:], matches)
		}
	}
}

func (tl *TopicLevel) match(segs [][]byte, matches *[]*TopicLevel) *TopicLevel {
	if (len(tl.Subscriptions) != 0 || len(tl.Children) == 0) && len(segs) == 0 || (gob.Equal(tl.Bytes, TOPIC_SINGLE_LEVEL_WILDCARD) && len(tl.Children) == 0 && len(segs) == 0) || gob.Equal(tl.Bytes, TOPIC_MULTI_LEVEL_WILDCARD) {
		*matches = append(*matches, tl)
		return tl
	}

	if len(tl.Children) != 0 && len(segs) != 0 {
		if tl.hasMultiWildcardAsChild && len(segs) == 0 {
			*matches = append(*matches, tl)
		}
		hits := tl.FindAll(segs[0])
		for _, hit := range hits {
			hit.match(segs[1:], matches)
		}
	}

	return nil
}

func (tl *TopicLevel) Retain(msg *Message) {
	tl.RetainedMessage = msg
}

func (tl *TopicLevel) AddChild(child *TopicLevel) {
	tl.Children = append(tl.Children, child)
}

func (tl *TopicLevel) ParseChildren(client *Client, children [][]byte, qos byte) {
	childrenLen := len(children)
	if childrenLen == 0 {
		return
	}

	b := children[0]
	l := tl.Find(b)
	if l == nil {
		l = &TopicLevel{Bytes: b, parent: tl}
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

func (tl *TopicLevel) ParseChildrenRetain(msg *Message, children [][]byte) {
	childrenLen := len(children)
	if childrenLen == 0 {
		return
	}

	b := children[0]
	l := tl.Find(b)
	if l == nil {
		l = &TopicLevel{Bytes: b, parent: tl}
		tl.AddChild(l)
	}

	if childrenLen == 1 && !gob.Equal(l.Bytes, TOPIC_SINGLE_LEVEL_WILDCARD) {
		l.Retain(msg)
		return
	}
	l.ParseChildrenRetain(msg, children[1:])
}

func (tl *TopicLevel) TraverseDelete(client *Client, children [][]byte) {
	childrenLen := len(children)
	if childrenLen == 0 {
		return
	}

	if l := tl.Find(children[0]); l != nil {
		if childrenLen == 1 {
			l.DeleteSubscription(client, true)
			return
		}

		l.TraverseDelete(client, children[1:])
	}
}

func (tl *TopicLevel) TraverseDeleteAll(client *Client) {
	for _, l := range tl.Children {
		l.DeleteSubscription(client, false)
		l.TraverseDeleteAll(client)
	}
}

func (tl *TopicLevel) Find(child []byte) *TopicLevel {
	for _, c := range tl.Children {
		if gob.Equal(c.Bytes, child) {
			return c
		}
	}
	return nil
}

func (tl *TopicLevel) FindAll(b []byte) (matches []*TopicLevel) {
	targetFound := false
	singleWildcardFound := false
	multiWildcardFound := false

	for _, f := range tl.Children {
		if !targetFound && gob.Equal(f.Bytes, b) {
			matches = append(matches, f)
			targetFound = true
		}
		if tl.hasSingleWildcardAsChild && !singleWildcardFound && gob.Equal(f.Bytes, TOPIC_SINGLE_LEVEL_WILDCARD) {
			matches = append(matches, f)
			singleWildcardFound = true
		}

		if tl.hasMultiWildcardAsChild && !multiWildcardFound && gob.Equal(f.Bytes, TOPIC_MULTI_LEVEL_WILDCARD) {
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
		if sub.Session.Id == client.ClientId {
			sub.QoS = qos

			if sub.Session.client == nil {
				sub.Session.client = client
			}
			//if tl.RetainedMessage != nil {
			//	GOTT.PublishRetained(tl.RetainedMessage, sub)
			//}
			return
		}
	}

	sub := &Subscription{
		Session: client.Session,
		QoS:     qos,
	}

	tl.Subscriptions = append(tl.Subscriptions, sub)
	//if tl.RetainedMessage != nil {
	//	GOTT.PublishRetained(tl.RetainedMessage, sub)
	//}
}

func (tl *TopicLevel) DeleteSubscription(client *Client, graceful bool) {
	for i, sub := range tl.Subscriptions {
		if sub.Session.Id == client.ClientId {
			if graceful || client.Session.clean {
				tl.deleteSubscription(i)
			} else {
				sub.Session.client = nil
			}
			return
		}
	}
}

func (tl *TopicLevel) Print(add string) {
	retained := "NONE"
	if tl.RetainedMessage != nil {
		retained = string(tl.RetainedMessage.Payload)
	}
	fmt.Println(add, tl.String(), "- subscriptions:", tl.SubscriptionsString(), "- retained:", retained)
	for _, c := range tl.Children {
		c.Print(add + indent)
	}
}

func (tl *TopicLevel) SubscriptionsString() string {
	strs := make([]string, 0)
	for _, s := range tl.Subscriptions {
		strs = append(strs, fmt.Sprintf("%v:%v", s.Session.client.ClientId, s.QoS))
	}

	return strings.Join(strs, ", ")
}

func (tl *TopicLevel) Name() string {
	return string(tl.Bytes)
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
	Filters              []*TopicLevel
	hasGlobalFilter      bool // has "#"
	hasTopSingleWildcard bool // has "+" as top level
}

func (ts *TopicStorage) AddTopLevel(tl *TopicLevel) {
	ts.Filters = append(ts.Filters, tl)
	if gob.Equal(tl.Bytes, TOPIC_MULTI_LEVEL_WILDCARD) {
		ts.hasGlobalFilter = true
	} else if gob.Equal(tl.Bytes, TOPIC_SINGLE_LEVEL_WILDCARD) {
		ts.hasTopSingleWildcard = true
	}
}

func (ts *TopicStorage) Find(b []byte) *TopicLevel {
	for _, f := range ts.Filters {
		if gob.Equal(f.Bytes, b) {
			return f
		}
	}
	return nil
}

func (ts *TopicStorage) FindAll(b []byte) (matches []*TopicLevel) {
	targetFound := false
	globalFilterFound := false
	topSingleWildcardFound := false

	for _, f := range ts.Filters {
		if !targetFound && gob.Equal(f.Bytes, b) {
			matches = append(matches, f)
			targetFound = true
		}
		if ts.hasGlobalFilter && !globalFilterFound && gob.Equal(f.Bytes, TOPIC_MULTI_LEVEL_WILDCARD) {
			matches = append(matches, f)
			globalFilterFound = true
		}
		if ts.hasTopSingleWildcard && !topSingleWildcardFound && gob.Equal(f.Bytes, TOPIC_SINGLE_LEVEL_WILDCARD) {
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
	fmt.Println("Topic Tree:")
	for _, f := range ts.Filters {
		f.Print("")
	}
}

func (ts *TopicStorage) Match(topic []byte) []*TopicLevel {
	matches := make([]*TopicLevel, 0)

	segs := gob.Split(topic, TOPIC_DELIM)

	hits := ts.FindAll(segs[0])

	//log.Printf("hits: %s", hits)

	for _, hit := range hits {
		hit.match(segs[1:], &matches)
	}

	return matches
}

func (ts *TopicStorage) ReverseMatch(filter []byte) []*TopicLevel {
	//if !utils.ByteInSlice(TOPIC_SINGLE_LEVEL_WILDCARD[0], filter) && !utils.ByteInSlice(TOPIC_MULTI_LEVEL_WILDCARD[0], filter) {
	//return nil
	//}

	segs := gob.Split(filter, TOPIC_DELIM)
	segsLen := len(segs)

	var matches []*TopicLevel

	topLevel := segs[0]
	isSingleWildcard := gob.Equal(topLevel, TOPIC_SINGLE_LEVEL_WILDCARD)
	isMultiWildcard := gob.Equal(topLevel, TOPIC_MULTI_LEVEL_WILDCARD)

	for _, level := range ts.Filters {
		if segsLen == 1 && isSingleWildcard && level.RetainedMessage != nil {
			matches = append(matches, level)
		} else if (isSingleWildcard || gob.Equal(topLevel, level.Bytes)) && !gob.Equal(level.Bytes, TOPIC_MULTI_LEVEL_WILDCARD) {
			level.reverse(segs[1:], &matches)
		} else if isMultiWildcard {
			level.reverse([][]byte{topLevel}, &matches)
		}
	}

	return matches
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

func ValidTopicName(topicName []byte) bool {
	return gob.IndexByte(topicName, TOPIC_MULTI_LEVEL_WILDCARD[0]) == -1 && gob.IndexByte(topicName, TOPIC_SINGLE_LEVEL_WILDCARD[0]) == -1 && len(topicName) > 0
}
