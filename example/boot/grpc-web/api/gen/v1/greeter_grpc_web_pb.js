/**
 * @fileoverview gRPC-Web generated client stub for api.v1
 * @enhanceable
 * @public
 */

// GENERATED CODE -- DO NOT EDIT!


/* eslint-disable */
// @ts-nocheck



const grpc = {};
grpc.web = require('grpc-web');

const proto = {};
proto.api = {};
proto.api.v1 = require('./greeter_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?grpc.web.ClientOptions} options
 * @constructor
 * @struct
 * @final
 */
proto.api.v1.GreeterClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options.format = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

};


/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?grpc.web.ClientOptions} options
 * @constructor
 * @struct
 * @final
 */
proto.api.v1.GreeterPromiseClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options.format = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.api.v1.GreeterRequest,
 *   !proto.api.v1.GreeterResponse>}
 */
const methodDescriptor_Greeter_Greeter = new grpc.web.MethodDescriptor(
  '/api.v1.Greeter/Greeter',
  grpc.web.MethodType.UNARY,
  proto.api.v1.GreeterRequest,
  proto.api.v1.GreeterResponse,
  /**
   * @param {!proto.api.v1.GreeterRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.api.v1.GreeterResponse.deserializeBinary
);


/**
 * @param {!proto.api.v1.GreeterRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.RpcError, ?proto.api.v1.GreeterResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.api.v1.GreeterResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.api.v1.GreeterClient.prototype.greeter =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/api.v1.Greeter/Greeter',
      request,
      metadata || {},
      methodDescriptor_Greeter_Greeter,
      callback);
};


/**
 * @param {!proto.api.v1.GreeterRequest} request The
 *     request proto
 * @param {?Object<string, string>=} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.api.v1.GreeterResponse>}
 *     Promise that resolves to the response
 */
proto.api.v1.GreeterPromiseClient.prototype.greeter =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/api.v1.Greeter/Greeter',
      request,
      metadata || {},
      methodDescriptor_Greeter_Greeter);
};


module.exports = proto.api.v1;

