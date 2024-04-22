package eventdistributor_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sharnoff/eventdistributor"
)

type MyEvent struct {
	id int
}

func TestDistibutor(t *testing.T) {
	var options eventdistributor.Options[MyEvent]
	var sizeChanges []int
	options.OnBufsizeChange(func(size int) {
		sizeChanges = append(sizeChanges, size)
	})
	var submitted []MyEvent
	options.OnSubmit(func(e MyEvent) {
		submitted = append(submitted, e)
	})
	var consumed []MyEvent
	options.OnFullyConsumed(func(e MyEvent) {
		consumed = append(consumed, e)
	})

	distributor := eventdistributor.New(options)

	t.Log("submit w/o consumers should be immediately considered consumed")
	s1 := distributor.Submit(MyEvent{id: 1})
	require.Equal(t, 1, len(submitted))
	require.Equal(t, 1, len(consumed))
	nowReady(t, s1)

	t.Log("immediately after subscribe, no events are ready")
	r1 := distributor.Subscribe()
	c1 := r1.WaitChan()
	nowNotReady(t, c1)
	r2 := distributor.Subscribe()
	c2 := r2.WaitChan()
	nowNotReady(t, c2)

	t.Log("submit 2")
	t.Log("after submit, size changes but nothing is consumed")
	s2 := distributor.Submit(MyEvent{id: 2})
	require.Equal(t, 1, len(sizeChanges))
	require.Equal(t, 2, len(submitted))
	require.Equal(t, 1, len(consumed))
	nowNotReady(t, s2)

	t.Log("after submit, readers are ready")
	ready(t, r1)
	nowReady(t, c1)
	ready(t, r2)
	nowReady(t, c2)

	t.Log("consumed is only after all readers consume")
	e := r1.Consume()
	require.Equal(t, 2, e.id)
	require.Equal(t, 1, len(consumed))
	notReady(t, r1)
	ready(t, r2)
	nowNotReady(t, s2)
	e = r2.Consume()
	require.Equal(t, 2, e.id)
	require.Equal(t, 2, len(consumed))
	notReady(t, r2)
	nowReady(t, s2)

	t.Log("Add another consumer")
	r3 := distributor.Subscribe()
	notReady(t, r1)
	notReady(t, r2)
	notReady(t, r3)

	t.Log("submit 3")
	t.Log("only one consumer reads")
	s3 := distributor.Submit(MyEvent{id: 3})
	ready(t, r1)
	ready(t, r2)
	ready(t, r3)
	e = r1.Consume()
	notReady(t, r1)
	require.Equal(t, 3, e.id)
	require.Equal(t, 2, len(consumed))
	nowNotReady(t, s3)

	t.Log("submit 4")
	t.Log("two consumers read")
	s4 := distributor.Submit(MyEvent{id: 4})
	ready(t, r1)
	e = r1.Consume()
	require.Equal(t, 4, e.id)
	notReady(t, r1)
	e = r2.Consume()
	require.Equal(t, 3, e.id)
	ready(t, r2)
	require.Equal(t, 2, len(consumed))
	nowNotReady(t, s3)
	nowNotReady(t, s4)

	t.Log("three consumers read")
	distributor.Submit(MyEvent{id: 5})
	ready(t, r1)
	e = r1.Consume()
	require.Equal(t, 5, e.id)
	notReady(t, r1)
	e = r2.Consume()
	require.Equal(t, 4, e.id)
	ready(t, r2)
	require.Equal(t, 2, len(consumed))
	e = r3.Consume()
	require.Equal(t, 3, e.id)
	ready(t, r3)
	require.Equal(t, 3, len(consumed))
	nowReady(t, s3)
	nowNotReady(t, s4)

	t.Log("new reader doesn't see pending stuff")
	r4 := distributor.Subscribe()
	notReady(t, r4)

	t.Log("events are considered consumed when reader unsubscribes")
	r3.Unsubscribe()
	require.Equal(t, 4, len(consumed))
	nowReady(t, s4)
}

func notReady(t *testing.T, reader eventdistributor.Reader[MyEvent]) {
	nowNotReady(t, reader.WaitChan())
}

func nowNotReady(t *testing.T, c <-chan struct{}) {
	select {
	case <-c:
		require.True(t, false)
	default:
	}
}

func ready(t *testing.T, reader eventdistributor.Reader[MyEvent]) {
	nowReady(t, reader.WaitChan())
}

func nowReady(t *testing.T, c <-chan struct{}) {
	select {
	case <-c:
	default:
		require.True(t, false)
	}
}
