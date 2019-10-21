package processchief

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/nordicdyno/processchief/pb"
)

func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		contentType := r.Header.Get("Content-Type")
		log.Printf("grpcHandlerFunc: r.ProtoMajor=%v, method=%v, contentType=%v\n",
			r.ProtoMajor, r.Method, contentType)

		if r.ProtoMajor == 2 && strings.Contains(contentType, "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}

func (cs *ControlServer) init(ctx context.Context) {
	if cs.server != nil {
		return
	}

	l, err := net.Listen("tcp", "localhost:5556")
	if err != nil {
		panic(err)
	}
	fmt.Printf("process chief connection addr: %s\n", l.Addr())

	grpcServer := grpc.NewServer()
	pb.RegisterControlAPIServer(grpcServer, cs)
	reflection.Register(grpcServer)

	mux := http.NewServeMux()
	gwmux := runtime.NewServeMux()

	mux.Handle("/", gwmux)

	dopts := []grpc.DialOption{grpc.WithInsecure()}
	err = pb.RegisterControlAPIHandlerFromEndpoint(ctx, gwmux, l.Addr().String(), dopts)

	cs.server = &serverWithContext{
		ctx:      ctx,
		listener: l,
		http: &http.Server{
			Handler: grpcHandlerFunc(grpcServer, mux),
		},
	}
}

func (cs *ControlServer) Start(ctx context.Context) error {
	cs.init(ctx)
	cs.Stop(ctx)

	cs.fin = make(chan struct{})

	return cs.server.http.Serve(cs.server.listener)
}

func (cs *ControlServer) stop(ctx context.Context) {
	// <-ctx.Done()
	log.Println("shutting down http...")
	err := cs.server.http.Shutdown(ctx)
	if err != nil {
		log.Printf("Shutdown error: %v", err)
	}
	close(cs.fin)
}

func (cs *ControlServer) Stop(ctx context.Context) {
	if cs.fin == nil {
		return
	}
	cs.stop(ctx)
	<-cs.fin
}
