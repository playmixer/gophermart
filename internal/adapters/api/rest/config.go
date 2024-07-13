package rest

type Config struct {
	Address string `env:"RUN_ADDRESS" envDefault:":8080"`
	Secret  string `env:"SECRET_KEY" envDefault:"secret_key"`
}
