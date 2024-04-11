package eventdistributor

import (
	"sync"
)

type EventDistributor[T any] struct {
	mu sync.Mutex

	basePosition int64
	buf          []eventInfo[T]

	nextRefcount int64
	waiters      chan struct{}

	onBufsizeChange []func(size int)
	onSubmit        []func(item T)
	onFullyConsumed []func(item T)
}

type eventInfo[T any] struct {
	refcount int64
	value    T
}

// NewEventDistributor creates a new EventDistributor with the provided options.
//
// If you don't have any options to set, the zero value of an EventDistributor is also valid.
func NewEventDistributor[T any](options ...DistributorOptions[T]) *EventDistributor[T] {
	d := &EventDistributor[T]{
		mu:              sync.Mutex{},
		basePosition:    0,
		buf:             nil,
		nextRefcount:    0,
		waiters:         nil,
		onBufsizeChange: nil,
		onSubmit:        nil,
		onFullyConsumed: nil,
	}

	for _, os := range options {
		for _, f := range os.modify {
			f(d)
		}
	}

	return d
}

func runCallbacks[T any](fs []func(T), v T) {
	for _, f := range fs {
		f(v)
	}
}

// Submit adds an event to the queue, notifying any waiting EventReaders
func (d *EventDistributor[T]) Submit(value T) {
	d.mu.Lock()
	defer d.mu.Unlock()

	runCallbacks(d.onSubmit, value)

	// If there's no readers waiting, then we should immediately discard the event.
	if len(d.buf) == 0 && d.nextRefcount == 0 {
		runCallbacks(d.onFullyConsumed, value)
		return
	}

	d.buf = append(d.buf, eventInfo[T]{
		refcount: d.nextRefcount,
		value:    value,
	})
	d.nextRefcount = 0
	if d.waiters != nil {
		close(d.waiters)
		d.waiters = nil
	}

	runCallbacks(d.onBufsizeChange, len(d.buf))
}

// Subscribe creates a new EventReader to receive future events from the EventDistributor.
//
// It is STRONGLY recommended to defer (*EventReader[T]).Unsubscribe() immediately after
// subscribing.
func (d *EventDistributor[T]) Subscribe() EventReader[T] {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.nextRefcount += 1
	return EventReader[T]{
		d:        d,
		position: d.basePosition + int64(len(d.buf)),
	}
}

type EventReader[T any] struct {
	d        *EventDistributor[T]
	position int64
}

var closedChannel <-chan struct{} = func() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

// WaitChan returns a channel that will be closed once there is an event that this EventReader has
// not yet seen.
func (r *EventReader[T]) WaitChan() <-chan struct{} {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()

	if r.position-r.d.basePosition < int64(len(r.d.buf)) {
		return closedChannel
	} else {
		if r.d.waiters == nil {
			r.d.waiters = make(chan struct{})
		}
		return r.d.waiters
	}
}

// Consume returns the first event that has not yet been seen by this Reader, marking it as "seen"
// so that the next call to WaitChan() will require a newer event.
func (r *EventReader[T]) Consume() T {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()

	idx := int(r.position - r.d.basePosition)
	value := r.d.buf[idx].value
	r.d.buf[idx].refcount -= 1

	if idx+1 < len(r.d.buf) {
		r.d.buf[idx+1].refcount += 1
	} else {
		r.d.nextRefcount += 1
	}

	r.d.cleanupOldEvents()
	return value
}

// Unsubscribe de-registers the EventReader, freeing any buffered events that may have been kept for
// it.
//
// If you stop using an EventReader and never call Unsubscribe, unread events will slowly
// accumulate, increasing the memory usage of your program.
func (r *EventReader[T]) Unsubscribe() {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()

	idx := int(r.position - r.d.basePosition)
	if idx < len(r.d.buf) {
		r.d.buf[idx].refcount -= 1
		if idx == 0 {
			r.d.cleanupOldEvents()
		}
	} else {
		r.d.nextRefcount -= 1
	}

	// For safety, remove the EventDistributor pointer so that future calls to Unsubscribe() will
	// panic, rather than silently corrupt the buffer.
	r.d = nil
}

func (d *EventDistributor[T]) cleanupOldEvents() {
	if len(d.buf) == 0 {
		return
	}

	firstNonEmpty := 0

	for ; firstNonEmpty < len(d.buf); firstNonEmpty += 1 {
		if d.buf[firstNonEmpty].refcount != 0 {
			break
		} else {
			runCallbacks(d.onFullyConsumed, d.buf[firstNonEmpty].value)
		}
	}

	if firstNonEmpty == 0 {
		return
	}

	if firstNonEmpty == len(d.buf) {
		d.buf = nil
	} else {
		d.buf = d.buf[firstNonEmpty:]
	}
	d.basePosition += int64(firstNonEmpty)

	runCallbacks(d.onBufsizeChange, len(d.buf))
}
