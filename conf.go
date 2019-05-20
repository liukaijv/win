package win

type Config struct {
	WorkerPoolSize uint32
	WorkerTaskMax  uint32
	AllowedOrigins []string
	MaxConn        int
}

var config Config

func init() {
	config = defaultConfig()
}

func defaultConfig() Config {

	return Config{
		WorkerPoolSize: 0,
		WorkerTaskMax:  1024,
		AllowedOrigins: []string{},
		MaxConn:        5000,
	}
}

func SetConfig(conf Config) {
	config = conf
}
