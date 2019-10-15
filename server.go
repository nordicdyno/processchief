package supervisor

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

	"github.com/nordicdyno/simple-hypervisor/pb"
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

func (s *ControlServer) init(ctx context.Context) {
	if s.server != nil {
		return
	}

	l, err := net.Listen("tcp", "localhost:5556")
	if err != nil {
		panic(err)
	}
	fmt.Printf("common addr: %s\n", l.Addr())

	grpcServer := grpc.NewServer()
	pb.RegisterServicesAPIServer(grpcServer, s)
	reflection.Register(grpcServer)

	mux := http.NewServeMux()
	gwmux := runtime.NewServeMux()

	mux.Handle("/", gwmux)

	dopts := []grpc.DialOption{grpc.WithInsecure()}
	err = pb.RegisterServicesAPIHandlerFromEndpoint(ctx, gwmux, l.Addr().String(), dopts)

	s.server = &serverWithContext{
		ctx:      ctx,
		listener: l,
		http: &http.Server{
			Handler: grpcHandlerFunc(grpcServer, mux),
		},
	}
}

func (s *ControlServer) Start(ctx context.Context) error {
	s.init(ctx)
	s.Stop(ctx)

	s.fin = make(chan struct{})

	return s.server.http.Serve(s.server.listener)
}

func (s *ControlServer) stop(ctx context.Context) {
	// <-ctx.Done()
	log.Println("shutting down http...")
	err := s.server.http.Shutdown(ctx)
	if err != nil {
		log.Printf("Shutdown error: %v", err)
	}
	close(s.fin)
}

func (s *ControlServer) Stop(ctx context.Context) {
	if s.fin == nil {
		return
	}
	s.stop(ctx)
	<-s.fin
}
