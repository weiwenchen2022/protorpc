package helloworld

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-netrpc_out=. --go-netrpc_opt=paths=source_relative ./helloworld.proto
