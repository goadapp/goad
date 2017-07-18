package infrastructure

const DefaultRunnerAsset = "data/lambda.zip"

type Infrastructure interface {
	Setup(settings Settings) (teardown func(), err error)
	Run(args InvokeArgs)
	GetQueueURL() string
}

type Settings struct {
	RunnerPath string
}

type InvokeArgs struct {
	File string   `json:"file"`
	Args []string `json:"args"`
}
