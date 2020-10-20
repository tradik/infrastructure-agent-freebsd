package server

type Config struct {
	CollectorBaseURL string
}

func NewConfig() Config {
	return Config{
		CollectorBaseURL: "https://staging-infra-api.newrelic.com",
	}
}
