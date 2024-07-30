package main

import (
	"context"
	"flag"
	"os"

	serverApp "github.com/zvfkjytytw/gophmarkt/internal/server/app"
)

const (
	envRunAddress    = "RUN_ADDRESS"
	envDatabaseURI   = "DATABASE_URI"
	envAccuralSystem = "ACCRUAL_SYSTEM_ADDRESS"
	envMigrationDir  = "MIGRATION_DIR"
)

func main() {
	var (
		configFile    string
		runAddress    string
		databaseURI   string
		accrualSystem string
		migrationDir  string
	)

	flag.StringVar(&configFile, "c", "../../build/server.yaml", "server config file")
	flag.StringVar(&runAddress, "a", "localhost:8080", "address and port of the service launch")
	flag.StringVar(&databaseURI, "d", "", "address of the database connection")
	flag.StringVar(&accrualSystem, "r", "", "address of the accrual calculation system")
	flag.StringVar(&migrationDir, "m", "../../build/migrations", "directory with the migration files")
	flag.Parse()

	value, ok := os.LookupEnv(envRunAddress)
	if ok {
		runAddress = value
	}

	value, ok = os.LookupEnv(envDatabaseURI)
	if ok {
		databaseURI = value
	}

	value, ok = os.LookupEnv(envAccuralSystem)
	if ok {
		accrualSystem = value
	}

	value, ok = os.LookupEnv(envMigrationDir)
	if ok {
		migrationDir = value
	}

	app, err := serverApp.NewApp(
		runAddress,
		databaseURI,
		accrualSystem,
	)
	if err != nil {
		panic(err)
	}

	// app, err := serverApp.NewAppFromFile(configFile)
	// if err != nil {
	// 	panic(err)
	// }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.Run(ctx)
}
