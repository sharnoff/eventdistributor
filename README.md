# eventdistributor

Internal event distribution package for Go.

The implementation uses signalling channels to notify waiting "readers", which can then "consume" an
event.

Events are buffered until all readers have seen them.

The implementation is rather small - feel free to read there for more information.
