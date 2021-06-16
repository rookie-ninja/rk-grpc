package rkgrpcextension

import (
	rkgrpcbasic "github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWithEntryNameAndType_HappyCase(t *testing.T) {
	opt := WithEntryNameAndType("ut-name", "ut-type")

	set := &optionSet{}

	opt(set)

	assert.Equal(t, "ut-name", set.EntryName)
	assert.Equal(t, "ut-type", set.EntryType)
}

func TestWithPrefix_HappyCase(t *testing.T) {
	opt := WithPrefix("ut-prefix")

	set := &optionSet{}

	opt(set)

	assert.Equal(t, "ut-prefix", set.Prefix)
}

func TestExtensionInterceptor_WithoutOption(t *testing.T) {
	UnaryServerInterceptor()

	assert.NotEmpty(t, optionsMap)
}

func TestExtensionInterceptor_HappyCase(t *testing.T) {
	UnaryServerInterceptor(WithEntryNameAndType("ut-name", "ut-type"))

	assert.NotNil(t, optionsMap[rkgrpcbasic.ToOptionsKey("ut-name", rkgrpcbasic.RpcTypeUnaryServer)])
}
