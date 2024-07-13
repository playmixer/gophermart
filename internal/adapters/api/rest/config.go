package rest

type Config struct {
	Address string `env:"RUN_ADDRESS"`
	Secret  string `env:"SECRET_KEY" envDefault:"secret_key"`
}
