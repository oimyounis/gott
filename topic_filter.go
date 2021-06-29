package gott

import (
	gob "bytes"
	"fmt"
	"gott/utils"
	"strings"
	"sync"
)

var (
	indent                   = "  "
	topicDelim               = []byte{47} // /
	topicSingleLevelWildcard = []byte{43} // +
	topicMultiLevelWildcard  = []byte{35} // #
)

type topicLevel struct {
	hasSingleWildcardAsChild atomicBool
	hasMultiWildcardAsChild  atomicBool
	parent                   *topicLevel
	Bytes                    []byte
	Children                 []*topicLevel
	Subscriptions            *subscriptionList
	QoS                      byte
	RetainedMessage          *message
}

func (tl *topicLevel) reverse(segs [][]byte, matches *[]*topicLevel) {
	if len(segs) == 0 {
		if tl.RetainedMessage != nil {
			*matches = append(*matches, tl)
		}
		return
	}

	seg := segs[0]

	isSingleWildcard := gob.Equal(seg, topicSingleLevelWildcard)
	isMultiWildcard := gob.Equal(seg, topicMultiLevelWildcard)

	if isMultiWildcard && !gob.Equal(tl.Bytes, topicMultiLevelWildcard) && tl.RetainedMessage != nil {
		*matches = append(*matches, tl)
	}

	for _, child := range tl.Children {
		if isMultiWildcard {
			child.reverse([][]byte{topicMultiLevelWildcard}, matches)
		} else if isSingleWildcard || gob.Equal(seg, child.Bytes) {
			child.reverse(segs[1:], matches)
		}
	}
}

func (tl *topicLevel) match(segs [][]byte, matches *[]*topicLevel) *topicLevel {
	if (tl.Subscriptions.Len() != 0 || len(tl.Children) == 0) && len(segs) == 0 || (gob.Equal(tl.Bytes, topicSingleLevelWildcard) && len(tl.Children) == 0 && len(segs) == 0) || gob.Equal(tl.Bytes, topicMultiLevelWildcard) {
		*matches = append(*matches, tl)
		return tl
	}

	if len(tl.Children) != 0 && len(segs) != 0 {
		if tl.hasMultiWildcardAsChild.Load() && len(segs) == 0 {
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
		l = &topicLevel{Bytes: b, parent: tl, Subscriptions: &subscriptionList{}}
		tl.addChild(l)
	}

	if gob.Equal(b, topicSingleLevelWildcard) {
		tl.hasSingleWildcardAsChild.Store(true)
	} else if gob.Equal(b, topicMultiLevelWildcard) {
		tl.hasMultiWildcardAsChild.Store(true)
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
		l = &topicLevel{Bytes: b, parent: tl, Subscriptions: &subscriptionList{}}
		tl.addChild(l)
	}

	if childrenLen == 1 && !gob.Equal(l.Bytes, topicSingleLevelWildcard) {
		l.retain(msg)
		return
	}
	l.parseChildrenRetain(msg, children[1:])
}

func (tl *topicLevel) traverseDelete(client *Client, children [][]byte) bool {
	childrenLen := len(children)
	if childrenLen == 0 {
		return false
	}

	if l := tl.find(children[0]); l != nil {
		if childrenLen == 1 {
			return l.DeleteSubscription(client, true)
		}

		return l.traverseDelete(client, children[1:])
	}

	return false
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
		if tl.hasSingleWildcardAsChild.Load() && !singleWildcardFound && gob.Equal(f.Bytes, topicSingleLevelWildcard) {
			matches = append(matches, f)
			singleWildcardFound = true
		}

		if tl.hasMultiWildcardAsChild.Load() && !multiWildcardFound && gob.Equal(f.Bytes, topicMultiLevelWildcard) {
			matches = append(matches, f)
			multiWildcardFound = true
		}

		if targetFound && (!tl.hasSingleWildcardAsChild.Load() || singleWildcardFound) && (!tl.hasMultiWildcardAsChild.Load() || multiWildcardFound) {
			break
		}
	}
	return
}

func (tl *topicLevel) createOrUpdateSubscription(client *Client, qos byte) {
	var ret bool
	tl.Subscriptions.Range(func(i int, sub *subscription) bool {
		if sub.Session.ID == client.ClientID {
			sub.QoS = qos

			sub.Session = client.Session
			ret = true
			return false
		}
		return true
	})
	if ret {
		return
	}

	sub := &subscription{
		Session: client.Session,
		QoS:     qos,
	}

	tl.Subscriptions.Add(sub)
}

// DeleteSubscription removes a client's subscription from the Topic Level.
func (tl *topicLevel) DeleteSubscription(client *Client, graceful bool) (success bool) {
	tl.Subscriptions.RangeDelete(func(i int, sub *subscription, delete func(int)) bool {
		if sub.Session.ID == client.ClientID {
			if graceful || client.Session.clean {
				delete(i)
			} else {
				sub.Session.client = nil
			}
			success = true
			return false
		}
		return true
	})
	return
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

	tl.Subscriptions.Range(func(i int, sub *subscription) bool {
		strs = append(strs, fmt.Sprintf("%v:%v", sub.Session.ID, sub.QoS))
		return true
	})

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
	mutex                sync.Mutex
	hasGlobalFilter      atomicBool // has "#"
	hasTopSingleWildcard atomicBool // has "+" as top level
}

func (ts *topicStorage) addTopLevel(tl *topicLevel) {
	ts.mutex.Lock()
	ts.Filters = append(ts.Filters, tl)
	ts.mutex.Unlock()
	if gob.Equal(tl.Bytes, topicMultiLevelWildcard) {
		ts.hasGlobalFilter.Store(true)
	} else if gob.Equal(tl.Bytes, topicSingleLevelWildcard) {
		ts.hasTopSingleWildcard.Store(true)
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

	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	for _, f := range ts.Filters {
		if !targetFound && gob.Equal(f.Bytes, b) {
			matches = append(matches, f)
			targetFound = true
		}
		if ts.hasGlobalFilter.Load() && !globalFilterFound && gob.Equal(f.Bytes, topicMultiLevelWildcard) {
			matches = append(matches, f)
			globalFilterFound = true
		}
		if ts.hasTopSingleWildcard.Load() && !topSingleWildcardFound && gob.Equal(f.Bytes, topicSingleLevelWildcard) {
			matches = append(matches, f)
			topSingleWildcardFound = true
		}

		if targetFound && (!ts.hasGlobalFilter.Load() || globalFilterFound) && (!ts.hasTopSingleWildcard.Load() || topSingleWildcardFound) {
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

// String returns the whole Topic Tree in string form.
func (ts *topicStorage) String() string {
	var topics []string

	for _, f := range ts.Filters {
		f.str(&topics)
	}

	return strings.Join(topics, "\n")
}

func (tl *topicLevel) str(topics *[]string) {
	if tl.Subscriptions.Len() > 0 {
		*topics = append(*topics, tl.String())
	}
	for _, c := range tl.Children {
		c.str(topics)
	}
}

func (ts *topicStorage) match(topic []byte) []*topicLevel {
	matches := make([]*topicLevel, 0)

	segs := gob.Split(topic, topicDelim)
	hits := ts.findAll(segs[0])

	for _, hit := range hits {
		hit.match(segs[1:], &matches)
	}

	return matches
}

func (ts *topicStorage) reverseMatch(filter []byte) []*topicLevel {
	segs := gob.Split(filter, topicDelim)
	segsLen := len(segs)

	var matches []*topicLevel

	topLevel := segs[0]
	isSingleWildcard := gob.Equal(topLevel, topicSingleLevelWildcard)
	isMultiWildcard := gob.Equal(topLevel, topicMultiLevelWildcard)

	for _, level := range ts.Filters {
		if segsLen == 1 && isSingleWildcard && level.RetainedMessage != nil {
			matches = append(matches, level)
		} else if (isSingleWildcard || gob.Equal(topLevel, level.Bytes)) && !gob.Equal(level.Bytes, topicMultiLevelWildcard) {
			level.reverse(segs[1:], &matches)
		} else if isMultiWildcard {
			level.reverse([][]byte{topLevel}, &matches)
		}
	}

	return matches
}

func validFilter(filter []byte) bool {
	multiWildcard := gob.IndexByte(filter, topicMultiLevelWildcard[0])
	singleWildcards := utils.IndexAllByte(filter, topicSingleLevelWildcard[0])
	filterLen := len(filter)

	if filterLen == 0 {
		return false
	}

	if gob.Count(filter, topicMultiLevelWildcard) > 1 || multiWildcard != -1 && multiWildcard != filterLen-1 || (multiWildcard > 0 && filter[multiWildcard-1] != topicDelim[0]) {
		return false
	}

	for _, idx := range singleWildcards {
		if idx > 0 && filter[idx-1] != topicDelim[0] {
			return false
		} else if filterLen > 1 && idx == 0 && idx != filterLen-1 && filter[idx+1] != topicDelim[0] {
			return false
		}
	}

	return true
}

func validTopicName(topicName []byte) bool {
	return gob.IndexByte(topicName, topicMultiLevelWildcard[0]) == -1 && gob.IndexByte(topicName, topicSingleLevelWildcard[0]) == -1 && len(topicName) > 0
}
