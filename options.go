package fastrpc

type Options struct {
}

type Option func(opts *Options)

func SetOption(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}
	return opts
}
