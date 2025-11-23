// -----------------------------------------------------------------------------
// Conduit CLI Tool - Laravel Artisan-Inspired Command Line Interface
// -----------------------------------------------------------------------------
// Bu CLI aracı, Laravel'in Artisan komutlarına benzer şekilde çalışır.
//
// Kullanım:
//   conduit <command> [arguments] [flags]
//
// Komutlar:
//   make:controller    - Controller oluşturur
//   make:model         - Model oluşturur
//   make:middleware    - Middleware oluşturur
//   make:job           - Job oluşturur
//   make:event         - Event oluşturur
//   make:listener      - Event Listener oluşturur
//   migrate            - Veritabanı migration'larını çalıştırır
//   migrate:rollback   - Son migration'ı geri alır
//   migrate:fresh      - Tüm tabloları siler ve migration'ları tekrar çalıştırır
//   migrate:status     - Migration durumunu gösterir
//   cache:clear        - Cache'i temizler
//   cache:forget       - Belirli bir cache key'ini siler
//   queue:work         - Queue worker başlatır
//   queue:listen       - Queue listener başlatır
//   queue:restart      - Queue worker'ları yeniden başlatır
//   serve              - Development sunucusunu başlatır
//   help               - Yardım gösterir
// -----------------------------------------------------------------------------

package main

import (
	"flag"
	"fmt"
	"os"
)

const Version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "make:controller":
		handleMakeController(os.Args[2:])
	case "make:model":
		handleMakeModel(os.Args[2:])
	case "make:middleware":
		handleMakeMiddleware(os.Args[2:])
	case "make:job":
		handleMakeJob(os.Args[2:])
	case "make:event":
		handleMakeEvent(os.Args[2:])
	case "make:listener":
		handleMakeListener(os.Args[2:])
	case "migrate":
		handleMigrate(os.Args[2:])
	case "migrate:rollback":
		handleMigrateRollback(os.Args[2:])
	case "migrate:fresh":
		handleMigrateFresh(os.Args[2:])
	case "migrate:status":
		handleMigrateStatus(os.Args[2:])
	case "cache:clear":
		handleCacheClear(os.Args[2:])
	case "cache:forget":
		handleCacheForget(os.Args[2:])
	case "queue:work":
		handleQueueWork(os.Args[2:])
	case "queue:listen":
		handleQueueListen(os.Args[2:])
	case "queue:restart":
		handleQueueRestart(os.Args[2:])
	case "serve":
		handleServe(os.Args[2:])
	case "help", "--help", "-h":
		printHelp()
	case "version", "--version", "-v":
		fmt.Printf("Conduit CLI v%s\n", Version)
	default:
		fmt.Printf("❌ Unknown command: %s\n\n", command)
		printHelp()
		os.Exit(1)
	}
}

// printHelp displays help information.
func printHelp() {
	fmt.Println(`
╔══════════════════════════════════════════════════════════════════════╗
║                   CONDUIT CLI - Laravel-Inspired                     ║
║                          Version ` + Version + `                              ║
╚══════════════════════════════════════════════════════════════════════╝

USAGE:
  conduit <command> [arguments] [flags]

MAKE COMMANDS:
  make:controller <name>     Create a new controller
  make:model <name>          Create a new model
  make:middleware <name>     Create a new middleware
  make:job <name>            Create a new job
  make:event <name>          Create a new event
  make:listener <name>       Create a new event listener

MIGRATION COMMANDS:
  migrate                    Run database migrations
  migrate:rollback           Rollback the last migration
  migrate:fresh              Drop all tables and re-run migrations
  migrate:status             Show migration status

CACHE COMMANDS:
  cache:clear                Clear all cache
  cache:forget <key>         Remove specific cache key

QUEUE COMMANDS:
  queue:work                 Start queue worker
  queue:listen               Start queue listener
  queue:restart              Restart queue workers

OTHER COMMANDS:
  serve                      Start development server
  help                       Show this help message
  version                    Show version

EXAMPLES:
  conduit make:controller UserController
  conduit make:model User
  conduit migrate
  conduit serve --port=8080

For more information about a specific command:
  conduit <command> --help
`)
}

// -----------------------------------------------------------------------------
// Make Commands
// -----------------------------------------------------------------------------

func handleMakeController(args []string) {
	fs := flag.NewFlagSet("make:controller", flag.ExitOnError)
	resource := fs.Bool("resource", false, "Create a resource controller with CRUD methods")
	api := fs.Bool("api", false, "Create an API controller (no views)")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Println("❌ Controller name required")
		fmt.Println("Usage: conduit make:controller <name> [--resource] [--api]")
		os.Exit(1)
	}

	name := fs.Arg(0)
	generateController(name, *resource, *api)
}

func handleMakeModel(args []string) {
	fs := flag.NewFlagSet("make:model", flag.ExitOnError)
	migration := fs.Bool("migration", false, "Create a migration file along with the model")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Println("❌ Model name required")
		fmt.Println("Usage: conduit make:model <name> [--migration]")
		os.Exit(1)
	}

	name := fs.Arg(0)
	generateModel(name, *migration)
}

func handleMakeMiddleware(args []string) {
	if len(args) < 1 {
		fmt.Println("❌ Middleware name required")
		fmt.Println("Usage: conduit make:middleware <name>")
		os.Exit(1)
	}

	name := args[0]
	generateMiddleware(name)
}

func handleMakeJob(args []string) {
	if len(args) < 1 {
		fmt.Println("❌ Job name required")
		fmt.Println("Usage: conduit make:job <name>")
		os.Exit(1)
	}

	name := args[0]
	generateJob(name)
}

func handleMakeEvent(args []string) {
	if len(args) < 1 {
		fmt.Println("❌ Event name required")
		fmt.Println("Usage: conduit make:event <name>")
		os.Exit(1)
	}

	name := args[0]
	generateEvent(name)
}

func handleMakeListener(args []string) {
	fs := flag.NewFlagSet("make:listener", flag.ExitOnError)
	event := fs.String("event", "", "The event class the listener should handle")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Println("❌ Listener name required")
		fmt.Println("Usage: conduit make:listener <name> --event=<EventName>")
		os.Exit(1)
	}

	name := fs.Arg(0)
	generateListener(name, *event)
}

// -----------------------------------------------------------------------------
// Migration Commands
// -----------------------------------------------------------------------------

func handleMigrate(args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	step := fs.Int("step", 0, "Number of migrations to run")
	fs.Parse(args)

	runMigrations(*step)
}

func handleMigrateRollback(args []string) {
	fs := flag.NewFlagSet("migrate:rollback", flag.ExitOnError)
	step := fs.Int("step", 1, "Number of migrations to rollback")
	fs.Parse(args)

	rollbackMigrations(*step)
}

func handleMigrateFresh(args []string) {
	fmt.Println("⚠️  WARNING: This will drop all tables and re-run all migrations!")
	fmt.Print("Are you sure? (yes/no): ")

	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "yes" {
		fmt.Println("Operation cancelled")
		return
	}

	freshMigrations()
}

func handleMigrateStatus(args []string) {
	showMigrationStatus()
}

// -----------------------------------------------------------------------------
// Cache Commands
// -----------------------------------------------------------------------------

func handleCacheClear(args []string) {
	clearCache()
}

func handleCacheForget(args []string) {
	if len(args) < 1 {
		fmt.Println("❌ Cache key required")
		fmt.Println("Usage: conduit cache:forget <key>")
		os.Exit(1)
	}

	key := args[0]
	forgetCacheKey(key)
}

// -----------------------------------------------------------------------------
// Queue Commands
// -----------------------------------------------------------------------------

func handleQueueWork(args []string) {
	fs := flag.NewFlagSet("queue:work", flag.ExitOnError)
	queue := fs.String("queue", "default", "The queue to listen on")
	maxJobs := fs.Int("max-jobs", 0, "Maximum number of jobs to process")
	timeout := fs.Int("timeout", 60, "Job timeout in seconds")
	fs.Parse(args)

	startQueueWorker(*queue, *maxJobs, *timeout)
}

func handleQueueListen(args []string) {
	fs := flag.NewFlagSet("queue:listen", flag.ExitOnError)
	queue := fs.String("queue", "default", "The queue to listen on")
	fs.Parse(args)

	startQueueListener(*queue)
}

func handleQueueRestart(args []string) {
	restartQueueWorkers()
}

// -----------------------------------------------------------------------------
// Serve Command
// -----------------------------------------------------------------------------

func handleServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 8080, "Port to run the server on")
	host := fs.String("host", "localhost", "Host to run the server on")
	fs.Parse(args)

	startDevServer(*host, *port)
}
