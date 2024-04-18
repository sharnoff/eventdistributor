package eventdistributor

// Options contains a set of options for Distributor initialization.
//
// The zero value is safe to use.
type Options[T any] struct {
	modify []func(*Distributor[T])
}

// OnBufsizeChange adds a callback to the options that will be called whenever the number of items
// in the buffer changes.
//
// NOTE: This is typically called during the Distributor's Submit(), Consume(), and
// Unsubscribe().
func (o *Options[T]) OnBufsizeChange(callback func(size int)) {
	o.modify = append(o.modify, func(d *Distributor[T]) {
		d.onBufsizeChange = append(d.onBufsizeChange, callback)
	})
}

// OnSubmit adds a callback to the options that will be called whenever an item is submitted with
// (*Distributor[T]).Submit().
//
// In the edge case where an item is immediately ignored because there's no readers, OnSubmit will
// be called before OnfullyConsumed.
func (o *Options[T]) OnSubmit(callback func(item T)) {
	o.modify = append(o.modify, func(d *Distributor[T]) {
		d.onSubmit = append(d.onSubmit, callback)
	})
}

// OnFullyConsumed adds a callback to the options that will be called whenever an item is dropped
// from the buffer.
//
// NOTE: If there are no active subscribers, the callback will be called *during* the call to
// (*Distributor[T]).Submit().
func (o *Options[T]) OnFullyConsumed(callback func(item T)) {
	o.modify = append(o.modify, func(d *Distributor[T]) {
		d.onFullyConsumed = append(d.onFullyConsumed, callback)
	})
}
