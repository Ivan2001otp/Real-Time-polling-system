FROM golang:1.24-alpine

#set working directory
WORKDIR /app

#install the necessary dependencies.
RUN apk add --no-cache git curl

# copy go mod files
COPY go.mod go.sum ./

# download dependencies
RUN go mod download

# copy the source code
COPY . .

RUN go build -o main ./cmd/server

# expose port
EXPOSE 8080

# run the application
CMD ["./main"]