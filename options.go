package events

// DistributorOptions contains a set of options for EventDistributor initialization.
//
// The zero value is safe to use.
type DistributorOptions[T any] struct {
	modify []func(*EventDistributor[T])
}

// OnBufsizeChange adds a callback to the options that will be called whenever the number of items
// in the buffer changes.
//
// NOTE: This is typically called during the EventDistributor's Submit(), Consume(), and
// Unsubscribe().
func (o *DistributorOptions[T]) OnBufsizeChange(callback func(size int)) {
	o.modify = append(o.modify, func(d *EventDistributor[T]) {
		d.onBufsizeChange = append(d.onBufsizeChange, callback)
	})
}

// OnSubmit adds a callback to the options that will be called whenever an item is submitted with
// (*EventDistributor[T]).Submit().
//
// In the edge case where an item is immediately ignored because there's no readers, OnSubmit will
// be called before OnfullyConsumed.
func (o *DistributorOptions[T]) OnSubmit(callback func(item T)) {
	o.modify = append(o.modify, func(d *EventDistributor[T]) {
		d.onSubmit = append(d.onSubmit, callback)
	})
}

// OnFullyConsumed adds a callback to the options that will be called whenever an item is dropped
// from the buffer.
//
// NOTE: If there are no active subscribers, the callback will be called *during* the call to
// (*EventDistributor[T]).Submit().
func (o *DistributorOptions[T]) OnFullyConsumed(callback func(item T)) {
	o.modify = append(o.modify, func(d *EventDistributor[T]) {
		d.onFullyConsumed = append(d.onFullyConsumed, callback)
	})
}
