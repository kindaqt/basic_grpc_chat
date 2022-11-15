# Chat App

## Get Started

### 1. Start Server

```{}
go run chat_server/chat_server.go
```

### 2. Start Client

Start client with the command below. Start a bunch and chat with yourself!

```{}
go run chat_client/chat_client.go -name YOUR_USERNAME -address SERVER_ADDRESS
```

*NOTE: the `-name` and `-address` flags are optional.*

### Regenerate proto

- **--go_out**: out dir of first protobuf file
- **--go-opt**: generate files relative to source (source being the first protobuf file)
- **--go-grpc_out**: out dir of the second protobuf file
- **--go-grpc_opt**: generate files relative to source (source being the second protobuf file)

```{}
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       chat/chat.proto
```

## TODO

- Security
  - Session Tokens
  - [Auth: SSL](https://grpc.io/docs/guides/auth/)
- User Managment
  - RPC method for user creation
  - Store users in database with passwords and session tokens
