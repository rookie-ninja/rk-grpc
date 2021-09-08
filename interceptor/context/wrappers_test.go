package rkgrpcctx

import (
	"context"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"testing"
)

type FakeServerStream struct {
	ctx context.Context
}

func (f FakeServerStream) SetHeader(md metadata.MD) error {
	return nil
}

func (f FakeServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (f FakeServerStream) SetTrailer(md metadata.MD) {
	return
}

func (f FakeServerStream) Context() context.Context {
	return f.ctx
}

func (f FakeServerStream) SendMsg(m interface{}) error {
	return nil
}

func (f FakeServerStream) RecvMsg(m interface{}) error {
	return nil
}

func TestWrapServerStream(t *testing.T) {
	ctx := context.TODO()

	// For WrapServerStream
	wrap := WrapServerStream(&FakeServerStream{
		ctx: ctx,
	})
	assert.Equal(t, wrap, WrapServerStream(&FakeServerStream{
		ctx: ctx,
	}))

	// For non WrapServerStream
	wrap = WrapServerStream(&FakeServerStream{
		ctx: ctx,
	})
	assert.Equal(t, ctx, wrap.Context())
}
