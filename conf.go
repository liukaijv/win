package win

type globalConfig struct {
	workerPoolSize uint32
	workerTaskMax  uint32
	allowedOrigins []string
	maxConn        int
}

var config globalConfig

func init() {
	config = defaultConfig()
}

func defaultConfig() globalConfig {

	return globalConfig{
		workerPoolSize: 0,
		workerTaskMax:  1024,
		allowedOrigins: []string{},
		maxConn:        5000,
	}
}

func SetPoolSize(pollSize uint32) {
	config.workerPoolSize = pollSize
}

func SetTaskSize(taskSize uint32) {
	config.workerTaskMax = taskSize
}

func SetAllowedOrigins(origins []string) {
	config.allowedOrigins = origins
}

func SetMaxConn(maxConn int) {
	config.maxConn = maxConn
}
