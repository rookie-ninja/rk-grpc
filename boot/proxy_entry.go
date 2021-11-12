// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

// Experimental. This is used as grpc proxy server which forwarding grpc request to backend grpc server if not implemented.
// Currently, grpc-gateway and grpcurl is not supported. The grpc client called from code is supported.
package rkgrpc

import (
	"context"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"math/rand"
	"net"
	"regexp"
	"time"
)

const (
	// ProxyEntryType default entry type
	ProxyEntryType = "ProxyEntry"
	// ProxyEntryNameDefault default entry name
	ProxyEntryNameDefault = "ProxyDefault"
	// ProxyEntryDescription default entry description
	ProxyEntryDescription = "Internal RK entry which implements proxy with Grpc framework."
	// HeaderBased header based proxy pattern
	HeaderBased = "headerBased"
	// PathBased grpc path(method) based proxy pattern
	PathBased = "pathBased"
	// IpBased remote IP based proxy pattern
	IpBased = "ipBased"
)

// BootConfigProxy Boot config which is for proxy entry.
//
// 1: Enabled: Enable prom entry.
// 2: Rules: Provide rules for proxying.
type BootConfigProxy struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	Rules   []struct {
		Type        string   `yaml:"type" json:"type"`
		HeaderPairs []string `yaml:"headerPairs" json:"headerPairs"`
		Dest        []string `yaml:"dest" json:"dest"`
		Paths       []string `yaml:"paths" json:"paths"`
		Ips         []string `yaml:"ips" json:"ips"`
	} `yaml:"rules" json:"rules"`
}

type rule struct {
	HeaderPattern []*HeaderPattern
	PathPattern   []*PathPattern
	IpPattern     []*IpPattern
	rand          *rand.Rand
}

// NewRule create a new proxy rules with options.
func NewRule(opts ...ruleOption) *rule {
	r := &rule{
		HeaderPattern: make([]*HeaderPattern, 0),
		PathPattern:   make([]*PathPattern, 0),
		IpPattern:     make([]*IpPattern, 0),
		rand:          rand.New(rand.NewSource(time.Now().Unix())),
	}

	for i := range opts {
		opts[i](r)
	}

	return r
}

type ruleOption func(*rule)

// WithHeaderPatterns provide header based patterns.
func WithHeaderPatterns(pattern ...*HeaderPattern) ruleOption {
	return func(r *rule) {
		r.HeaderPattern = append(r.HeaderPattern, pattern...)
	}
}

// WithPathPatterns provide path based patterns.
func WithPathPatterns(pattern ...*PathPattern) ruleOption {
	return func(r *rule) {
		r.PathPattern = append(r.PathPattern, pattern...)
	}
}

// WithIpPatterns provide IP based patterns.
func WithIpPatterns(pattern ...*IpPattern) ruleOption {
	return func(r *rule) {
		r.IpPattern = append(r.IpPattern, pattern...)
	}
}

// HeaderPattern defines proxy rules based on header.
//
// Proxy will validate headers in metadata with provided rules.
type HeaderPattern struct {
	Headers map[string]string
	Dest    []string
}

// PathPattern defines proxy rules based on path.
//
// The incoming path should match with rules.
// Path rule support regex.
type PathPattern struct {
	Paths []string
	Dest  []string
}

// IpPattern defines proxy rules based on remote IPs.
//
// Ip rule support CIDR.
type IpPattern struct {
	Cidrs []string
	Dest  []string
}

// Incoming remote IP should match user defined CIDR.
func (r *rule) matchIpPattern(ctx context.Context) (bool, string) {
	remoteIp, _, _ := rkgrpcinter.GetRemoteAddressSet(ctx)

	// iterate pattern slice
	for i := range r.IpPattern {
		pattern := r.IpPattern[i]

		// iterate CIDR
		for j := range pattern.Cidrs {
			cidr := pattern.Cidrs[j]
			_, subnet, err := net.ParseCIDR(cidr)
			if err != nil {
				continue
			}

			// match CIDR
			if subnet.Contains(net.ParseIP(remoteIp)) {
				return true, pattern.Dest[r.rand.Intn(len(pattern.Dest))]
			}
		}
	}

	return false, ""
}

// Incoming path should match user defined regex.
func (r *rule) matchPathPattern(ctx context.Context) (bool, string) {
	method, ok := grpc.Method(ctx)

	if !ok {
		return false, ""
	}

	// iterate pattern slice
	for i := range r.PathPattern {
		pattern := r.PathPattern[i]

		// iterate path
		for j := range pattern.Paths {
			pathRegex := pattern.Paths[j]

			// match regex
			if matched, err := regexp.MatchString(pathRegex, method); err == nil && matched {
				return true, pattern.Dest[r.rand.Intn(len(pattern.Dest))]
			}
		}
	}

	return false, ""
}

// Incoming header should match user defined rule.
func (r *rule) matchHeaderPattern(ctx context.Context) (bool, string) {
	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return false, ""
	}

	// iterate pattern slice
	for i := range r.HeaderPattern {
		pattern := r.HeaderPattern[i]

		matched := true
		// iterate header
		for k, v1 := range pattern.Headers {
			// all the headers must be exists in metadata
			v2, _ := md[k]
			matched = containsSlice(v2, v1)

			if !matched {
				break
			}
		}

		if matched {
			return true, pattern.Dest[r.rand.Intn(len(pattern.Dest))]
		}

	}

	return false, ""
}

func containsSlice(src []string, target string) bool {
	if src == nil {
		return false
	}

	for i := range src {
		if src[i] == target {
			return true
		}
	}
	return false
}

// GetDirector creates a default Director based on rules.
func (r *rule) GetDirector() Director {
	return func(ctx context.Context) (context.Context, *grpc.ClientConn, error) {
		// check ip pattern
		if matched, dest := r.matchIpPattern(ctx); matched && len(dest) > 0 {
			conn, err := grpc.DialContext(ctx, dest,
				grpc.WithInsecure(),
				grpc.WithDefaultCallOptions(grpc.ForceCodec(Codec())))

			return ctx, conn, err
		}

		// check path pattern
		if matched, dest := r.matchPathPattern(ctx); matched && len(dest) > 0 {
			conn, err := grpc.DialContext(ctx, dest,
				grpc.WithInsecure(),
				grpc.WithDefaultCallOptions(grpc.ForceCodec(Codec())))
			return ctx, conn, err
		}

		// check header pattern
		if matched, dest := r.matchHeaderPattern(ctx); matched && len(dest) > 0 {
			conn, err := grpc.DialContext(ctx, dest,
				grpc.WithInsecure(),
				grpc.WithDefaultCallOptions(grpc.ForceCodec(Codec())))
			return ctx, conn, err
		}

		return nil, nil, status.Errorf(codes.Unimplemented, "Unknown method")
	}
}

type ProxyEntry struct {
	EntryName        string                    `json:"entryName" yaml:"entryName"`
	EntryType        string                    `json:"entryType" yaml:"entryType"`
	EntryDescription string                    `json:"entryDescription" yaml:"entryDescription"`
	ZapLoggerEntry   *rkentry.ZapLoggerEntry   `json:"zapLoggerEntry" yaml:"zapLoggerEntry"`
	EventLoggerEntry *rkentry.EventLoggerEntry `json:"eventLoggerEntry" yaml:"eventLoggerEntry"`
	r                *rule                     `json:"-" yaml:"-"`
}

// ProxyEntryOption Proxy entry option used while initializing proxy entry via code
type ProxyEntryOption func(*ProxyEntry)

// WithNameProm Name of proxy entry
func WithNameProxy(name string) ProxyEntryOption {
	return func(entry *ProxyEntry) {
		entry.EntryName = name
	}
}

// WithZapLoggerEntryProxy rkentry.ZapLoggerEntry of proxy entry
func WithZapLoggerEntryProxy(zapLoggerEntry *rkentry.ZapLoggerEntry) ProxyEntryOption {
	return func(entry *ProxyEntry) {
		entry.ZapLoggerEntry = zapLoggerEntry
	}
}

// WithEventLoggerEntryProxy rkentry.EventLoggerEntry of proxy entry
func WithEventLoggerEntryProxy(eventLoggerEntry *rkentry.EventLoggerEntry) ProxyEntryOption {
	return func(entry *ProxyEntry) {
		entry.EventLoggerEntry = eventLoggerEntry
	}
}

// WithRuleProxy Provide rule
func WithRuleProxy(r *rule) ProxyEntryOption {
	return func(entry *ProxyEntry) {
		entry.r = r
	}
}

// NewProxyEntry Create a proxy entry with options
func NewProxyEntry(opts ...ProxyEntryOption) *ProxyEntry {
	entry := &ProxyEntry{
		EventLoggerEntry: rkentry.GlobalAppCtx.GetEventLoggerEntryDefault(),
		ZapLoggerEntry:   rkentry.GlobalAppCtx.GetZapLoggerEntryDefault(),
		EntryName:        ProxyEntryNameDefault,
		EntryType:        ProxyEntryType,
		EntryDescription: ProxyEntryDescription,
	}

	for i := range opts {
		opts[i](entry)
	}

	if entry.ZapLoggerEntry == nil {
		entry.ZapLoggerEntry = rkentry.GlobalAppCtx.GetZapLoggerEntryDefault()
	}

	if entry.EventLoggerEntry == nil {
		entry.EventLoggerEntry = rkentry.GlobalAppCtx.GetEventLoggerEntryDefault()
	}

	return entry
}

// Bootstrap Start prometheus client
func (entry *ProxyEntry) Bootstrap(ctx context.Context) {
	event := entry.EventLoggerEntry.GetEventHelper().Start(
		"bootstrap",
		rkquery.WithEntryName(entry.EntryName),
		rkquery.WithEntryType(entry.EntryType))

	logger := entry.ZapLoggerEntry.GetLogger()

	if raw := ctx.Value(bootstrapEventIdKey); raw != nil {
		event.SetEventId(raw.(string))
		logger = logger.With(zap.String("eventId", event.GetEventId()))
	}

	entry.logBasicInfo(event)

	defer entry.EventLoggerEntry.GetEventHelper().Finish(event)

	logger.Info("Bootstrapping proxyEntry.", event.ListPayloads()...)
}

// Interrupt Shutdown prometheus client
func (entry *ProxyEntry) Interrupt(ctx context.Context) {
	event := entry.EventLoggerEntry.GetEventHelper().Start(
		"interrupt",
		rkquery.WithEntryName(entry.EntryName),
		rkquery.WithEntryType(entry.EntryType))

	logger := entry.ZapLoggerEntry.GetLogger()

	if raw := ctx.Value(bootstrapEventIdKey); raw != nil {
		event.SetEventId(raw.(string))
		logger = logger.With(zap.String("eventId", event.GetEventId()))
	}

	entry.logBasicInfo(event)

	defer entry.EventLoggerEntry.GetEventHelper().Finish(event)

	logger.Info("Interrupting proxyEntry.", event.ListPayloads()...)
}

// GetName Return name of proxy entry
func (entry *ProxyEntry) GetName() string {
	return entry.EntryName
}

// GetType Return type of prom entry
func (entry *ProxyEntry) GetType() string {
	return entry.EntryType
}

// GetDescription Get description of entry
func (entry *ProxyEntry) GetDescription() string {
	return entry.EntryDescription
}

// String Stringfy prom entry
func (entry *ProxyEntry) String() string {
	bytes, _ := json.Marshal(entry)
	return string(bytes)
}

// MarshalJSON Marshal entry
func (entry *ProxyEntry) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"entryName":        entry.EntryName,
		"entryType":        entry.EntryType,
		"entryDescription": entry.EntryDescription,
		"eventLoggerEntry": entry.EventLoggerEntry.GetName(),
		"zapLoggerEntry":   entry.ZapLoggerEntry.GetName(),
	}

	return json.Marshal(&m)
}

// UnmarshalJSON Unmarshal entry
func (entry *ProxyEntry) UnmarshalJSON(b []byte) error {
	return nil
}

// Log basic info into event
func (entry *ProxyEntry) logBasicInfo(event rkquery.Event) {
	event.AddPayloads(
		zap.String("entryName", entry.EntryName),
		zap.String("entryType", entry.EntryType),
	)
}

// *************************************
// *************** Codec ***************
// *************************************

// Codec returns a proxying grpc.Codec with the default protobuf codec as parent.
//
// See CodecWithParent.
func Codec() encoding.Codec {
	return CodecWithFallback(&protoCodec{})
}

// CodecWithParent returns a proxying grpc.Codec with a user provided codec as parent.
//
// This codec is *crucial* to the functioning of the proxy. It allows the proxy server to be oblivious
// to the schema of the forwarded messages. It basically treats a gRPC message frame as raw bytes.
// However, if the server handler, or the client caller are not proxy-internal functions it will fall back
// to trying to decode the message using a fallback codec.
func CodecWithFallback(fallback encoding.Codec) encoding.Codec {
	return &rawCodec{fallback}
}

type rawCodec struct {
	parentCodec encoding.Codec
}

type frame struct {
	payload []byte
}

// Marshal rawCodec.
func (r *rawCodec) Marshal(v interface{}) ([]byte, error) {
	out, ok := v.(*frame)
	if !ok {
		return r.parentCodec.Marshal(v)
	}
	return out.payload, nil

}

// Unmarshal rawCodec.
func (r *rawCodec) Unmarshal(data []byte, v interface{}) error {
	dst, ok := v.(*frame)
	if !ok {
		return r.parentCodec.Unmarshal(data, v)
	}
	dst.payload = data
	return nil
}

// Name return name of parent codec name.
func (r *rawCodec) Name() string {
	return r.parentCodec.Name()
}

// protoCodec is a Codec implementation with protobuf. It is the default rawCodec for gRPC.
type protoCodec struct{}

// Marshal protoCodec.
func (protoCodec) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}

// Unmarshal protoCodec.
func (protoCodec) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}

// Name return name of protoCodec.
func (protoCodec) Name() string {
	return "rk-proto-codec"
}

// ************************************
// ************* Handler **************
// ************************************

// Director creates context and connection based on proxy rules.
type Director func(context.Context) (context.Context, *grpc.ClientConn, error)

var clientStreamDescForProxying = &grpc.StreamDesc{
	ServerStreams: true,
	ClientStreams: true,
}

// TransparentHandler returns a handler that attempts to proxy all requests that are not registered in the server.
// The indented use here is as a transparent proxy, where the server doesn't know about the services implemented by the
// backends. It should be used as a `grpc.UnknownServiceHandler`.
//
// This can *only* be used if the `server` also uses grpcproxy.CodecForServer() ServerOption.
func TransparentHandler(director Director) grpc.StreamHandler {
	streamer := &handler{director}
	return streamer.handler
}

type handler struct {
	director Director
}

// handler is where the real magic of proxying happens.
// It is invoked like any gRPC server stream and uses the gRPC server framing to get and receive bytes from the wire,
// forwarding it to a ClientStream established against the relevant ClientConn.
func (s *handler) handler(srv interface{}, serverStream grpc.ServerStream) error {
	// little bit of gRPC internals never hurt anyone
	fullMethodName, ok := grpc.MethodFromServerStream(serverStream)

	if !ok {
		return status.Errorf(codes.Internal, "lowLevelServerStream not exists in context")
	}
	// We require that the director's returned context inherits from the serverStream.Context().
	outgoingCtx, backendConn, err := s.director(serverStream.Context())
	if err != nil {
		return err
	}

	clientCtx, clientCancel := context.WithCancel(outgoingCtx)
	clientCtx = metadata.AppendToOutgoingContext(clientCtx, "X-Forwarded-For", rkcommon.GetLocalIP())

	clientStream, err := grpc.NewClientStream(clientCtx, clientStreamDescForProxying, backendConn, fullMethodName)

	if err != nil {
		return err
	}
	// Explicitly *do not close* s2cErrChan and c2sErrChan, otherwise the select below will not terminate.
	// Channels do not have to be closed, it is just a control flow mechanism, see
	// https://groups.google.com/forum/#!msg/golang-nuts/pZwdYRGxCIk/qpbHxRRPJdUJ
	s2cErrChan := s.forwardServerToClient(serverStream, clientStream)
	c2sErrChan := s.forwardClientToServer(clientStream, serverStream)
	// We don't know which side is going to stop sending first, so we need a select between the two.
	for i := 0; i < 2; i++ {
		select {
		case s2cErr := <-s2cErrChan:
			if s2cErr == io.EOF {
				// this is the happy case where the sender has encountered io.EOF, and won't be sending anymore./
				// the clientStream>serverStream may continue pumping though.
				clientStream.CloseSend()
				break
			} else {
				// however, we may have gotten a receive error (stream disconnected, a read error etc) in which case we need
				// to cancel the clientStream to the backend, let all of its goroutines be freed up by the CancelFunc and
				// exit with an error to the stack
				clientCancel()
				return grpc.Errorf(codes.Internal, "failed proxying s2c: %v", s2cErr)
			}
		case c2sErr := <-c2sErrChan:
			// This happens when the clientStream has nothing else to offer (io.EOF), returned a gRPC error. In those two
			// cases we may have received Trailers as part of the call. In case of other errors (stream closed) the trailers
			// will be nil.
			serverStream.SetTrailer(clientStream.Trailer())
			// c2sErr will contain RPC error from client code. If not io.EOF return the RPC error as server stream error.
			if c2sErr != io.EOF {
				return c2sErr
			}
			return nil
		}
	}
	return status.Errorf(codes.Internal, "gRPC proxying should never reach this stage.")
}

func (s *handler) forwardClientToServer(src grpc.ClientStream, dst grpc.ServerStream) chan error {
	ret := make(chan error, 1)
	go func() {
		f := &frame{}
		for i := 0; ; i++ {
			if err := src.RecvMsg(f); err != nil {
				ret <- err // this can be io.EOF which is happy case
				break
			}
			if i == 0 {
				// This is a bit of a hack, but client to server headers are only readable after first client msg is
				// received but must be written to server stream before the first msg is flushed.
				// This is the only place to do it nicely.
				md, err := src.Header()
				if err != nil {
					ret <- err
					break
				}
				if err := dst.SendHeader(md); err != nil {
					ret <- err
					break
				}
			}
			if err := dst.SendMsg(f); err != nil {
				ret <- err
				break
			}
		}
	}()
	return ret
}

func (s *handler) forwardServerToClient(src grpc.ServerStream, dst grpc.ClientStream) chan error {
	ret := make(chan error, 1)
	go func() {
		f := &frame{}
		for i := 0; ; i++ {
			if err := src.RecvMsg(f); err != nil {
				ret <- err // this can be io.EOF which is happy case
				break
			}
			if err := dst.SendMsg(f); err != nil {
				ret <- err
				break
			}
		}
	}()
	return ret
}
