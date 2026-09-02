// Command 1trade is the 1Trade platform CLI — the primary engineer interface (F04). It is a thin
// client over the platform HTTP APIs (auth, ledger, inference); auth state lives in
// ~/.1trade/config.json. See `1trade help`.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/trade1/cli/internal/client"
	"github.com/trade1/cli/internal/config"
	"github.com/trade1/cli/internal/ui"
)

// Version is set at build time (-ldflags -X main.Version=...).
var Version = "dev"

const usage = `1trade — the 1Trade platform CLI

Usage: 1trade <command> [args]

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
  infer chat -m MODEL "prompt"            Run a chat completion (streams; --json for the object)
  chat [-m MODEL]                         Interactive multi-turn chat (slash commands, live cost)

GPU instances:
  gpu types                               List GPU types + price + availability
  gpu create --type h100 [--count N] [--image stable] [--region R]   Launch an instance
  gpu list [--state running]              List your instances
  gpu get <id>                            Show one instance (+ connection info)
  gpu stop <id>                           Stop an instance (frees its GPUs)
  gpu start <id>                          Restart a stopped instance
  gpu delete <id>                         Terminate an instance

Keys:
  keys create --name N                    Mint an API key (shown once)
  keys list                               List API keys
  keys revoke <id>                        Revoke a key

Config:
  config get | set <key> <value>          View/set platform URLs (platform_url|gateway_url|ledger_url|compute_url)
  version                                 Print the CLI version

Conventions:
  infer chat streams by default; add --json for the full object (scripting/CI).
  Destructive commands (gpu delete, keys revoke) confirm; --yes skips.
  Colour is on for a terminal and off when piped or under NO_COLOR.
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
	case "chat":
		err = cmdChat(cfg, args)
	case "gpu":
		err = cmdGPU(cfg, args)
	case "keys":
		err = cmdKeys(cfg, args)
	case "config":
		err = cmdConfig(cfg, args)
	case "version", "--version", "-v":
		fmt.Println("1trade " + Version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		fmt.Print(usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.Red("error:")+" "+err.Error())
		// Map upstream errors to a next step so the failure is a fork in the road, not a dead end.
		var ae *client.APIError
		if errors.As(err, &ae) {
			if hint := ui.Hint(ae.Status, ae.Code); hint != "" {
				fmt.Fprintln(os.Stderr, ui.Dim("  → "+hint))
			}
		}
		os.Exit(1)
	}
}
