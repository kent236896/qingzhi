package main

import (
	"fmt"
	"os"

	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/internal/platform/database"
	"qingzhi/enterprise-api/internal/platform/redact"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) != 1 || (args[0] != "up" && args[0] != "down" && args[0] != "version") {
		fmt.Fprintln(os.Stderr, "usage: migrate up|down|version")
		return 2
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %s\n", err)
		return 1
	}
	message, err := database.Migrate(cfg.Postgres, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %s\n", redact.Apply(err.Error(), cfg.Postgres.Secrets()...))
		return 1
	}
	fmt.Println(message)
	return 0
}
