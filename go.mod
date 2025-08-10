module github.com/mooncorn/nodelink/test

go 1.21

require github.com/mooncorn/nodelink/server v0.0.0

require github.com/mooncorn/nodelink/proto v0.0.0

replace github.com/mooncorn/nodelink/server => ./server

replace github.com/mooncorn/nodelink/proto => ./proto
