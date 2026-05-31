// Command exascale is the Exascale platform CLI — the primary engineer interface (F04). It is a thin
// client over the platform HTTP APIs (auth, ledger, inference); auth state lives in
// ~/.exascale/config.json. See `exascale help`.
package main

import (
	"fmt"
	"os"

	"github.com/exascale/cli/internal/config"
)

// Version is set at build time (-ldflags -X main.Version=...).
var Version = "dev"

const usage = `exascale — the Exascale platform CLI

Usage: exascale <command> [args]

Auth:
  login [--email E]                       Log in (saves a session token)
  signup [--email E] [--name N]           Create an account + log in
  whoami                                  Show the current identity
  logout                                  Clear the saved token

Credits:
  credits balance                         Show credit balances
  credits transactions                    Recent ledger transactions
  credits convert --from F --to T --amount N   Convert credits (rate + 1% spread)

Inference:
  catalog                                 List available models
  infer chat -m MODEL "prompt"            Run a chat completion

Keys:
  keys create --name N                    Mint an API key (shown once)
  keys list                               List API keys
  keys revoke <id>                        Revoke a key

Config:
  config get | set <key> <value>          View/set platform URLs (platform_url|gateway_url|ledger_url)
  version                                 Print the CLI version
`

// main dispatches the subcommand and prints a friendly error on failure.
func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		return
	}
	cfg := config.Load()
	args := os.Args[2:]
	var err error
	switch os.Args[1] {
	case "login":
		err = cmdLogin(cfg, args)
	case "signup":
		err = cmdSignup(cfg, args)
	case "whoami":
		err = cmdWhoami(cfg, args)
	case "logout":
		err = cmdLogout(cfg, args)
	case "credits":
		err = cmdCredits(cfg, args)
	case "catalog", "models":
		err = cmdCatalog(cfg, args)
	case "infer":
		err = cmdInfer(cfg, args)
	case "keys":
		err = cmdKeys(cfg, args)
	case "config":
		err = cmdConfig(cfg, args)
	case "version", "--version", "-v":
		fmt.Println("exascale " + Version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		fmt.Print(usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}
}
