GGPC_GW_DIR = $(shell go list -f '{{ .Dir }}' -m github.com/grpc-ecosystem/grpc-gateway)

# go get github.com/grpc-ecosystem/grpc-gateway
.PHONY: pb
pb:
	go list -f '{{ .Dir }}' -m github.com/grpc-ecosystem/grpc-gateway
	# echo "$(GGPC_GW_DIR)"
	protoc -I/usr/local/include -I./ \
		-I$(GGPC_GW_DIR)/third_party/googleapis \
		--go_out=plugins=grpc:./  \
		--grpc-gateway_out=logtostderr=true:. \
		--swagger_out=logtostderr=true:. \
		pb/*.proto