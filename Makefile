
compile-proto: 
	docker run --rm -v "$(PWD)":/proto -w "/proto" rvolosatovs/protoc --go_out=plugins=grpc:/proto --proto_path /proto proto/chittychat.proto

server:
	docker run -d -p 8080:8080 .