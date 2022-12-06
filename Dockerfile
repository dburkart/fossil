# ----- Build
FROM golang:1.19 AS build

RUN mkdir -p /go/src/github.com/dburkart/fossil
WORKDIR /go/src/github.com/dburkart/fossil
ADD . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o fossil main.go

# ----- Runtime
FROM scratch

WORKDIR /go/src/github.com/dburkart/fossil
COPY --from=build /go/src/github.com/dburkart/fossil/fossil /fossil
ENTRYPOINT ["/fossil"]