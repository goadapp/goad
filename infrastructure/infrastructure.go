package infrastructure

type Infrastructure interface {
	Setup() (teardown func(), err error)
	Run(args InvokeArgs)
	GetQueueURL() string
}

type InvokeArgs struct {
	File string   `json:"file"`
	Args []string `json:"args"`
}
