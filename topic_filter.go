package gott

import (
	gob "bytes"
	"fmt"
	"gott/utils"
	"strings"
)

var indent = "  "

type topicLevel struct {
	hasSingleWildcardAsChild bool
	hasMultiWildcardAsChild  bool
	parent                   *topicLevel
	Bytes                    []byte
	Children                 []*topicLevel
	Subscriptions            []*subscription
	QoS                      byte
	RetainedMessage          *message
}

func (tl *topicLevel) deleteSubscription(index int) {
	var newSubs []*subscription
	newSubs = append(newSubs, tl.Subscriptions[:index]...)
	if index != len(tl.Subscriptions)-1 {
		newSubs = append(newSubs, tl.Subscriptions[index+1:]...)
	}
	tl.Subscriptions = newSubs
}

func (tl *topicLevel) reverse(segs [][]byte, matches *[]*topicLevel) {
	if len(segs) == 0 {
		if !gob.Equal(tl.Bytes, TopicMultiLevelWildcard) && tl.RetainedMessage != nil {
			*matches = append(*matches, tl)
		}
		return
	}

	seg := segs[0]

	isSingleWildcard := gob.Equal(seg, TopicSingleLevelWildcard)
	isMultiWildcard := gob.Equal(seg, TopicMultiLevelWildcard)

	if isMultiWildcard && !gob.Equal(tl.Bytes, TopicMultiLevelWildcard) && tl.RetainedMessage != nil {
		*matches = append(*matches, tl)
	}

	for _, child := range tl.Children {
		if isMultiWildcard {
			child.reverse([][]byte{TopicMultiLevelWildcard}, matches)
		} else if isSingleWildcard || gob.Equal(seg, child.Bytes) {
			child.reverse(segs[1:], matches)
		}
	}
}

func (tl *topicLevel) match(segs [][]byte, matches *[]*topicLevel) *topicLevel {
	if (len(tl.Subscriptions) != 0 || len(tl.Children) == 0) && len(segs) == 0 || (gob.Equal(tl.Bytes, TopicSingleLevelWildcard) && len(tl.Children) == 0 && len(segs) == 0) || gob.Equal(tl.Bytes, TopicMultiLevelWildcard) {
		*matches = append(*matches, tl)
		return tl
	}

	if len(tl.Children) != 0 && len(segs) != 0 {
		if tl.hasMultiWildcardAsChild && len(segs) == 0 {
			*matches = append(*matches, tl)
		}
		hits := tl.findAll(segs[0])
		for _, hit := range hits {
			hit.match(segs[1:], matches)
		}
	}

	return nil
}

func (tl *topicLevel) retain(msg *message) {
	tl.RetainedMessage = msg
}

func (tl *topicLevel) addChild(child *topicLevel) {
	tl.Children = append(tl.Children, child)
}

func (tl *topicLevel) parseChildren(client *Client, children [][]byte, qos byte) {
	childrenLen := len(children)
	if childrenLen == 0 {
		return
	}

	b := children[0]
	l := tl.find(b)

	if l == nil {
		l = &topicLevel{Bytes: b, parent: tl}
		tl.addChild(l)
	}

	if gob.Equal(b, TopicSingleLevelWildcard) {
		tl.hasSingleWildcardAsChild = true
	} else if gob.Equal(b, TopicMultiLevelWildcard) {
		tl.hasMultiWildcardAsChild = true
		tl.createOrUpdateSubscription(client, qos)
	}

	if childrenLen == 1 {
		l.createOrUpdateSubscription(client, qos)
		return
	}
	l.parseChildren(client, children[1:], qos)
}

func (tl *topicLevel) parseChildrenRetain(msg *message, children [][]byte) {
	childrenLen := len(children)
	if childrenLen == 0 {
		return
	}

	b := children[0]
	l := tl.find(b)
	if l == nil {
		l = &topicLevel{Bytes: b, parent: tl}
		tl.addChild(l)
	}

	if childrenLen == 1 && !gob.Equal(l.Bytes, TopicSingleLevelWildcard) {
		l.retain(msg)
		return
	}
	l.parseChildrenRetain(msg, children[1:])
}

func (tl *topicLevel) traverseDelete(client *Client, children [][]byte) {
	childrenLen := len(children)
	if childrenLen == 0 {
		return
	}

	if l := tl.find(children[0]); l != nil {
		if childrenLen == 1 {
			l.DeleteSubscription(client, true)
			return
		}

		l.traverseDelete(client, children[1:])
	}
}

func (tl *topicLevel) traverseDeleteAll(client *Client) {
	for _, l := range tl.Children {
		l.DeleteSubscription(client, false)
		l.traverseDeleteAll(client)
	}
}

func (tl *topicLevel) find(child []byte) *topicLevel {
	for _, c := range tl.Children {
		if gob.Equal(c.Bytes, child) {
			return c
		}
	}
	return nil
}

func (tl *topicLevel) findAll(b []byte) (matches []*topicLevel) {
	targetFound := false
	singleWildcardFound := false
	multiWildcardFound := false

	for _, f := range tl.Children {
		if !targetFound && gob.Equal(f.Bytes, b) {
			matches = append(matches, f)
			targetFound = true
		}
		if tl.hasSingleWildcardAsChild && !singleWildcardFound && gob.Equal(f.Bytes, TopicSingleLevelWildcard) {
			matches = append(matches, f)
			singleWildcardFound = true
		}

		if tl.hasMultiWildcardAsChild && !multiWildcardFound && gob.Equal(f.Bytes, TopicMultiLevelWildcard) {
			matches = append(matches, f)
			multiWildcardFound = true
		}

		if targetFound && (!tl.hasSingleWildcardAsChild || singleWildcardFound) && (!tl.hasMultiWildcardAsChild || multiWildcardFound) {
			break
		}
	}
	return
}

func (tl *topicLevel) createOrUpdateSubscription(client *Client, qos byte) {
	for _, sub := range tl.Subscriptions {
		if sub.Session.ID == client.ClientID {
			sub.QoS = qos

			sub.Session = client.Session
			return
		}
	}

	sub := &subscription{
		Session: client.Session,
		QoS:     qos,
	}

	tl.Subscriptions = append(tl.Subscriptions, sub)
}

// DeleteSubscription removes a client's subscription from the Topic Level.
func (tl *topicLevel) DeleteSubscription(client *Client, graceful bool) {
	for i, sub := range tl.Subscriptions {
		if sub.Session.ID == client.ClientID {
			if graceful || client.Session.clean {
				tl.deleteSubscription(i)
			} else {
				sub.Session.client = nil
			}
			return
		}
	}
}

// Print outputs the Topic Level's path, subscriptions and the retained message in string form.
func (tl *topicLevel) Print(add string) {
	retained := "NONE"
	if tl.RetainedMessage != nil {
		retained = string(tl.RetainedMessage.Payload)
	}
	fmt.Println(add, tl.String(), "- subscriptions:", tl.subscriptionsString(), "- retained:", retained)
	for _, c := range tl.Children {
		c.Print(add + indent)
	}
}

func (tl *topicLevel) subscriptionsString() string {
	strs := make([]string, 0)
	for _, s := range tl.Subscriptions {
		strs = append(strs, fmt.Sprintf("%v:%v", s.Session.ID, s.QoS))
	}

	return strings.Join(strs, ", ")
}

// Name returns the name of the Topic Level.
func (tl *topicLevel) Name() string {
	return string(tl.Bytes)
}

// Path returns the path of the Topic Level.
func (tl *topicLevel) Path() string {
	if tl.parent != nil {
		return tl.parent.Path() + "/" + tl.Name()
	}
	return tl.Name()
}

// String converts to the string form of the Topic Level.
// Same as calling Path().
func (tl *topicLevel) String() string {
	return tl.Path()
}

type topicStorage struct {
	Filters              []*topicLevel
	hasGlobalFilter      bool // has "#"
	hasTopSingleWildcard bool // has "+" as top level
}

func (ts *topicStorage) addTopLevel(tl *topicLevel) {
	ts.Filters = append(ts.Filters, tl)
	if gob.Equal(tl.Bytes, TopicMultiLevelWildcard) {
		ts.hasGlobalFilter = true
	} else if gob.Equal(tl.Bytes, TopicSingleLevelWildcard) {
		ts.hasTopSingleWildcard = true
	}
}

func (ts *topicStorage) find(b []byte) *topicLevel {
	for _, f := range ts.Filters {
		if gob.Equal(f.Bytes, b) {
			return f
		}
	}
	return nil
}

func (ts *topicStorage) findAll(b []byte) (matches []*topicLevel) {
	targetFound := false
	globalFilterFound := false
	topSingleWildcardFound := false

	for _, f := range ts.Filters {
		if !targetFound && gob.Equal(f.Bytes, b) {
			matches = append(matches, f)
			targetFound = true
		}
		if ts.hasGlobalFilter && !globalFilterFound && gob.Equal(f.Bytes, TopicMultiLevelWildcard) {
			matches = append(matches, f)
			globalFilterFound = true
		}
		if ts.hasTopSingleWildcard && !topSingleWildcardFound && gob.Equal(f.Bytes, TopicSingleLevelWildcard) {
			matches = append(matches, f)
			topSingleWildcardFound = true
		}

		if targetFound && (!ts.hasGlobalFilter || globalFilterFound) && (!ts.hasTopSingleWildcard || topSingleWildcardFound) {
			break
		}
	}
	return
}

// Print outputs the whole Topic Tree to stdout.
func (ts *topicStorage) Print() {
	fmt.Println("Topic Tree:")
	for _, f := range ts.Filters {
		f.Print("")
	}
}

func (ts *topicStorage) match(topic []byte) []*topicLevel {
	matches := make([]*topicLevel, 0)

	segs := gob.Split(topic, TopicDelim)
	hits := ts.findAll(segs[0])

	for _, hit := range hits {
		hit.match(segs[1:], &matches)
	}

	return matches
}

func (ts *topicStorage) reverseMatch(filter []byte) []*topicLevel {
	segs := gob.Split(filter, TopicDelim)
	segsLen := len(segs)

	var matches []*topicLevel

	topLevel := segs[0]
	isSingleWildcard := gob.Equal(topLevel, TopicSingleLevelWildcard)
	isMultiWildcard := gob.Equal(topLevel, TopicMultiLevelWildcard)

	for _, level := range ts.Filters {
		if segsLen == 1 && isSingleWildcard && level.RetainedMessage != nil {
			matches = append(matches, level)
		} else if (isSingleWildcard || gob.Equal(topLevel, level.Bytes)) && !gob.Equal(level.Bytes, TopicMultiLevelWildcard) {
			level.reverse(segs[1:], &matches)
		} else if isMultiWildcard {
			level.reverse([][]byte{topLevel}, &matches)
		}
	}

	return matches
}

func validFilter(filter []byte) bool {
	multiWildcard := gob.IndexByte(filter, TopicMultiLevelWildcard[0])
	singleWildcards := utils.IndexAllByte(filter, TopicSingleLevelWildcard[0])
	filterLen := len(filter)

	if filterLen == 0 {
		return false
	}

	if gob.Count(filter, TopicMultiLevelWildcard) > 1 || multiWildcard != -1 && multiWildcard != filterLen-1 || (multiWildcard > 0 && filter[multiWildcard-1] != TopicDelim[0]) {
		return false
	}

	for _, idx := range singleWildcards {
		if idx > 0 && filter[idx-1] != TopicDelim[0] {
			return false
		} else if filterLen > 1 && idx == 0 && idx != filterLen-1 && filter[idx+1] != TopicDelim[0] {
			return false
		}
	}

	return true
}

func validTopicName(topicName []byte) bool {
	return gob.IndexByte(topicName, TopicMultiLevelWildcard[0]) == -1 && gob.IndexByte(topicName, TopicSingleLevelWildcard[0]) == -1 && len(topicName) > 0
}
