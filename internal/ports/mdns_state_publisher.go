package ports

import "context"

type MDNSStatePublisher interface {
	Publish(ctx context.Context, up, down []string) error
}
