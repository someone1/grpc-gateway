package main

import (
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func generateFromText(t *testing.T, input string) *plugin.CodeGeneratorResponse {
	msgTbl = make(map[string]*descriptor.DescriptorProto)
	var req plugin.CodeGeneratorRequest
	if err := proto.UnmarshalText(input, &req); err != nil {
		t.Fatalf("proto.Unmarshal(%q, &req) failed with %v; want success", input, err)
	}
	return generate(&req)
}

func mustGenerateFromText(t *testing.T, input string) []*plugin.CodeGeneratorResponse_File {
	resp := generateFromText(t, input)
	if resp.Error != nil {
		t.Fatalf("generate(%s) failed with %s", input, resp.GetError())
	}
	return resp.File
}

func testGenerate(t *testing.T, input string, outputs map[string]string) {
	var expected plugin.CodeGeneratorResponse
	for fname, content := range outputs {
		expected.File = append(expected.File, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(fname),
			Content: proto.String(content),
		})
	}
	resp := generateFromText(t, input)
	if !proto.Equal(resp, &expected) {
		t.Errorf("generate(%s) = %s; want %s", input, proto.MarshalTextString(resp), proto.MarshalTextString(&expected))
	}
}

func TestGenerateEmtpy(t *testing.T) {
	testGenerate(t, "", nil)
}

func TestGenerate(t *testing.T) {
	testGenerate(t, `
file_to_generate: "example.proto"
proto_file <
  name: "example.proto"
  package: "example"
  syntax: "proto3"
  message_type <
    name: "SimpleMessage"
    field <
      name: "id"
      number: 1
      label: LABEL_REQUIRED,
      type: TYPE_STRING
    >
  >
  service <
    name: "EchoService"
    method <
      name: "Echo"
      input_type: ".example.SimpleMessage"
      output_type: ".example.SimpleMessage"
      options <
        [gengo.grpc.gateway.ApiMethodOptions.api_options] <
          path: "/v1/example/echo/:id"
          method: "POST"
        >
      >
    >
    method <
      name: "EchoBody"
      input_type: ".example.SimpleMessage"
      output_type: ".example.SimpleMessage"
      options <
        [gengo.grpc.gateway.ApiMethodOptions.api_options] <
          path: "/v1/example/echo_body"
          method: "POST"
        >
      >
    >
  >
>`, map[string]string{
		"example.pb.gw.go": `// Code generated by protoc-gen-grpc-gateway
// source: example.proto
// DO NOT EDIT!

/*
Package example is a reverse proxy.

It translates gRPC into RESTful JSON APIs.
*/
package example

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gengo/grpc-gateway/runtime"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/zenazn/goji/web"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var _ codes.Code
var _ io.Reader
var _ = runtime.String

func request_EchoService_Echo(ctx context.Context, c web.C, client EchoServiceClient, req *http.Request) (msg proto.Message, err error) {
	var protoReq SimpleMessage

	var val string
	var ok bool

	val, ok = c.URLParams["id"]
	if !ok {
		return nil, grpc.Errorf(codes.InvalidArgument, "missing parameter %s", "id")
	}
	protoReq.Id, err = runtime.String(val)
	if err != nil {
		return nil, err
	}

	return client.Echo(ctx, &protoReq)
}

func request_EchoService_EchoBody(ctx context.Context, c web.C, client EchoServiceClient, req *http.Request) (msg proto.Message, err error) {
	var protoReq SimpleMessage

	if err = json.NewDecoder(req.Body).Decode(&protoReq); err != nil {
		return nil, err
	}

	return client.EchoBody(ctx, &protoReq)
}

// RegisterEchoServiceHandlerFromEndpoint is same as RegisterEchoServiceHandler but
// automatically dials to "endpoint" and closes the connection when "ctx" gets done.
func RegisterEchoServiceHandlerFromEndpoint(ctx context.Context, mux *web.Mux, endpoint string) (err error) {
	conn, err := grpc.Dial(endpoint)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if cerr := conn.Close(); cerr != nil {
				glog.Error("Failed to close conn to %s: %v", endpoint, cerr)
			}
			return
		}
		go func() {
			<-ctx.Done()
			if cerr := conn.Close(); cerr != nil {
				glog.Error("Failed to close conn to %s: %v", endpoint, cerr)
			}
		}()
	}()

	return RegisterEchoServiceHandler(ctx, mux, conn)
}

// RegisterEchoServiceHandler registers the http handlers for service EchoService to "mux".
// The handlers forward requests to the grpc endpoint over "conn".
func RegisterEchoServiceHandler(ctx context.Context, mux *web.Mux, conn *grpc.ClientConn) error {
	client := NewEchoServiceClient(conn)

	mux.Post("/v1/example/echo/:id", func(c web.C, w http.ResponseWriter, req *http.Request) {
		resp, err := request_EchoService_Echo(ctx, c, client, req)
		if err != nil {
			runtime.HTTPError(w, err)
			return
		}

		runtime.ForwardResponseMessage(w, resp)

	})

	mux.Post("/v1/example/echo_body", func(c web.C, w http.ResponseWriter, req *http.Request) {
		resp, err := request_EchoService_EchoBody(ctx, c, client, req)
		if err != nil {
			runtime.HTTPError(w, err)
			return
		}

		runtime.ForwardResponseMessage(w, resp)

	})

	return nil
}
`,
	})
}

func TestGenerateWithExternalMessage(t *testing.T) {
	input := `
file_to_generate: "example.proto"
proto_file <
  name: "github.com/example/proto/message.proto"
  package: "com.example.proto"
  message_type <
    name: "SimpleMessage"
    field <
      name: "id"
      number: 1
      label: LABEL_REQUIRED,
      type: TYPE_STRING
    >
  >
>
proto_file <
  name: "github.com/example/sub/proto/another.proto"
  package: "com.example.sub.proto"
  message_type <
    name: "AnotherMessage"
    field <
      name: "id"
      number: 1
      label: LABEL_REQUIRED,
      type: TYPE_STRING
    >
  >
>
proto_file <
  name: "example.proto"
  package: "example"
  dependency: "github.com/example/proto/message.proto"
  dependency: "github.com/example/sub/proto/another.proto"
  public_dependency: 0
  public_dependency: 1
  syntax: "proto3"
  service <
    name: "EchoService"
    method <
      name: "SimpleToAnother"
      input_type: ".com.example.proto.SimpleMessage"
      output_type: ".com.example.sub.proto.AnotherMessage"
      options <
        [gengo.grpc.gateway.ApiMethodOptions.api_options] <
          path: "/v1/example/conv/:id"
          method: "POST"
        >
      >
    >
    method <
      name: "AnotherToSimple"
      input_type: ".com.example.sub.proto.AnotherMessage"
      output_type: ".com.example.proto.SimpleMessage"
      options <
        [gengo.grpc.gateway.ApiMethodOptions.api_options] <
          path: "/v1/example/rconv/:id"
          method: "POST"
        >
      >
    >
  >
>`
	files := mustGenerateFromText(t, input)
	if got, want := len(files), 1; got != want {
		t.Errorf("len(generate(%s).File) = %d; want %d", got, want)
		return
	}
	content := files[0].GetContent()

	if want := `com_example_proto "github.com/example/proto"`; !strings.Contains(content, want) {
		t.Errorf("content = %s; want it to contain %s", content, want)
	}
	if want := `com_example_sub_proto "github.com/example/sub/proto"`; !strings.Contains(content, want) {
		t.Errorf("content = %s; want it to contain %s", content, want)
	}
}