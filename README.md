Generates a Go service based on protobuffer files.

This tool uses an existing proto buffer description and
* Reads the service
* Generates a go interface for the service
* Generates a handler to dispatch incoming binary messages to the service
* Handles de-/serialization of parameters, responses and errors
* Generates TypeScript code to call the service

Requirements
============

Missing parameters
------------------
Proto-Buffers don't support non-existing parameters or return types.
Hence, there needs to be one "void" type in the proto file that resembles missing parameters or response types.

E.g.:
```
message Void {
}
```

RPC vs Push Services
--------------------
To distinguish server side services (RPC: request client -> server -> response client) 
vs server initiates push messages (Server Side Push: server -> client),
The services need to be tagged with `is_rpc` and `is_ssp`.

This requires the option definition in the proto service file:

```
import "proto/resources.proto";

extend google.protobuf.ServiceOptions {
    optional bool is_rpc = 50000;
    optional bool is_ssp = 50001;
}
```

The service can be tagged like this:
```
service MyService {
    option (is_rpc) = true;
    ...
}
```
