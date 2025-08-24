module github.com/mooncorn/nodelink/test

go 1.23.5

toolchain go1.24.6

require github.com/mooncorn/nodelink/server v0.0.0

require github.com/mooncorn/nodelink/proto v0.0.0

replace github.com/mooncorn/nodelink/server => ./server

replace github.com/mooncorn/nodelink/proto => ./proto
