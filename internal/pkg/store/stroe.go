package store

import "context"

var client Factory

type Factory interface {
	Users() UserStore
	Ping(ctx context.Context) error
}

// Client return the store client instance.
func Client() Factory {
	return client
}

// SetClient set the iam store client.
func SetClient(factory Factory) {
	client = factory
}
