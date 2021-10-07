module github.com/shestakovda/journal

go 1.13

require (
	github.com/apple/foundationdb/bindings/go v0.0.0-20201222225940-f3aef311ccfb
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/flatbuffers v1.12.0
	github.com/google/go-cmp v0.5.6
	github.com/shestakovda/envx v1.0.1
	github.com/shestakovda/errx v1.2.0
	github.com/shestakovda/fdbx v0.3.9
	github.com/shestakovda/fdbx/v2 v2.0.2
	github.com/shestakovda/typex v1.0.1
	github.com/stretchr/testify v1.6.1
)

replace github.com/shestakovda/fdbx/v2 v2.0.2 => github.com/stejls/fdbx/v2 v2.0.4
