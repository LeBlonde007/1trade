package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/exascale/cli/internal/client"
	"github.com/exascale/cli/internal/config"
	"golang.org/x/term"
)

// readLine reads a trimmed line from stdin after printing a label.
func readLine(label string) string {
	fmt.Print(label)
	s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(s)
}

// readPassword reads a password without echoing it.
func readPassword(label string) string {
	fmt.Print(label)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// newIdempotencyKey returns a random key for a mutating call.
func newIdempotencyKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "cli_" + hex.EncodeToString(b)
}

// requireToken errors if the user isn't logged in.
func requireToken(cfg config.Config) error {
	if cfg.Token == "" {
		return errors.New("not logged in — run: exascale login")
	}
	return nil
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// cmdLogin authenticates and saves the token, then prints the identity.
func cmdLogin(cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	email := fs.String("email", "", "account email")
	password := fs.String("password", "", "account password (prefer the prompt or EXASCALE_PASSWORD)")
	_ = fs.Parse(args)
	e := firstNonEmpty(*email)
	if e == "" {
		e = readLine("Email: ")
	}
	p := firstNonEmpty(*password, os.Getenv("EXASCALE_PASSWORD"))
	if p == "" {
		p = readPassword("Password: ")
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := client.Do("POST", cfg.PlatformURL, "/v1/auth/login", "", nil, map[string]string{"email": e, "password": p}, &out); err != nil {
		return err
	}
	cfg.Token = out.Token
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Println("Logged in.")
	return cmdWhoami(cfg, nil)
}

// cmdSignup creates an account and saves the token.
func cmdSignup(cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("signup", flag.ExitOnError)
	email := fs.String("email", "", "account email")
	name := fs.String("name", "", "tenant name")
	_ = fs.Parse(args)
	e := firstNonEmpty(*email)
	if e == "" {
		e = readLine("Email: ")
	}
	p := readPassword("Password: ")
	var out struct {
		Token string `json:"token"`
	}
	if err := client.Do("POST", cfg.PlatformURL, "/v1/auth/signup", "", nil, map[string]string{"email": e, "password": p, "tenant_name": *name}, &out); err != nil {
		return err
	}
	cfg.Token = out.Token
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Println("Account created.")
	return cmdWhoami(cfg, nil)
}

// cmdWhoami prints the current identity.
func cmdWhoami(cfg config.Config, _ []string) error {
	if err := requireToken(cfg); err != nil {
		return err
	}
	var me map[string]any
	if err := client.Do("GET", cfg.PlatformURL, "/v1/auth/me", cfg.Token, nil, nil, &me); err != nil {
		return err
	}
	fmt.Printf("%-10s %v\n", "email", me["email"])
	fmt.Printf("%-10s %v\n", "tenant", me["tenant_id"])
	fmt.Printf("%-10s %v\n", "roles", me["roles"])
	fmt.Printf("%-10s %v\n", "is_paper", me["is_paper"])
	return nil
}

// cmdLogout clears the saved token.
func cmdLogout(cfg config.Config, _ []string) error {
	cfg.Token = ""
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Println("Logged out.")
	return nil
}

// cmdCredits dispatches the credits subcommands.
func cmdCredits(cfg config.Config, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: exascale credits balance|transactions|convert")
	}
	switch args[0] {
	case "balance":
		return creditsBalance(cfg)
	case "transactions":
		return creditsTransactions(cfg)
	case "convert":
		return creditsConvert(cfg, args[1:])
	default:
		return fmt.Errorf("unknown credits subcommand %q", args[0])
	}
}

// creditsBalance prints the tenant's credit balances.
func creditsBalance(cfg config.Config) error {
	if err := requireToken(cfg); err != nil {
		return err
	}
	var out struct {
		Balances []struct {
			CreditType   string `json:"credit_type"`
			Balance      string `json:"balance"`
			LockedAmount string `json:"locked_amount"`
		} `json:"balances"`
	}
	if err := client.Do("GET", cfg.LedgerURL, "/v1/credits/balances", cfg.Token, nil, nil, &out); err != nil {
		return err
	}
	if len(out.Balances) == 0 {
		fmt.Println("No credits yet.")
		return nil
	}
	fmt.Printf("%-14s %16s %16s\n", "CREDIT", "BALANCE", "LOCKED")
	for _, b := range out.Balances {
		fmt.Printf("%-14s %16s %16s\n", b.CreditType, b.Balance, b.LockedAmount)
	}
	return nil
}

// creditsTransactions prints recent ledger transactions.
func creditsTransactions(cfg config.Config) error {
	if err := requireToken(cfg); err != nil {
		return err
	}
	var out struct {
		Transactions []struct {
			CreditType   string `json:"credit_type"`
			Operation    string `json:"operation"`
			Amount       string `json:"amount"`
			BalanceAfter string `json:"balance_after"`
			CreatedAt    string `json:"created_at"`
		} `json:"transactions"`
	}
	if err := client.Do("GET", cfg.LedgerURL, "/v1/credits/transactions", cfg.Token, nil, nil, &out); err != nil {
		return err
	}
	fmt.Printf("%-22s %-12s %-9s %16s %16s\n", "TIME", "OP", "CREDIT", "AMOUNT", "BALANCE")
	for _, t := range out.Transactions {
		fmt.Printf("%-22s %-12s %-9s %16s %16s\n", t.CreatedAt, t.Operation, t.CreditType, t.Amount, t.BalanceAfter)
	}
	return nil
}

// creditsConvert converts between credit types.
func creditsConvert(cfg config.Config, args []string) error {
	if err := requireToken(cfg); err != nil {
		return err
	}
	fs := flag.NewFlagSet("convert", flag.ExitOnError)
	from := fs.String("from", "", "source credit type")
	to := fs.String("to", "", "target credit type")
	amount := fs.String("amount", "", "amount of from to convert")
	_ = fs.Parse(args)
	if *from == "" || *to == "" || *amount == "" {
		return errors.New("usage: exascale credits convert --from ai_index --to text --amount 100")
	}
	var out struct {
		Debit  map[string]any `json:"debit"`
		Credit map[string]any `json:"credit"`
	}
	if err := client.Do("POST", cfg.LedgerURL, "/v1/credits/convert", cfg.Token,
		map[string]string{"Idempotency-Key": newIdempotencyKey()},
		map[string]string{"from": *from, "to": *to, "amount": *amount}, &out); err != nil {
		return err
	}
	fmt.Printf("Converted %v %v → %v %v\n", out.Debit["amount"], out.Debit["credit_type"], out.Credit["amount"], out.Credit["credit_type"])
	return nil
}

// cmdCatalog lists the model catalog.
func cmdCatalog(cfg config.Config, _ []string) error {
	if err := requireToken(cfg); err != nil {
		return err
	}
	var out struct {
		Data []struct {
			ID       string `json:"id"`
			Exascale struct {
				Modality string `json:"modality"`
				Unit     string `json:"unit"`
				Price    string `json:"price"`
			} `json:"exascale"`
		} `json:"data"`
	}
	if err := client.Do("GET", cfg.GatewayURL, "/v1/models", cfg.Token, nil, nil, &out); err != nil {
		return err
	}
	fmt.Printf("%-20s %-10s %14s  %s\n", "MODEL", "MODALITY", "PRICE", "UNIT")
	for _, m := range out.Data {
		fmt.Printf("%-20s %-10s %14s  %s\n", m.ID, m.Exascale.Modality, m.Exascale.Price, m.Exascale.Unit)
	}
	return nil
}

// cmdInfer runs an inference command (chat).
func cmdInfer(cfg config.Config, args []string) error {
	if len(args) == 0 || args[0] != "chat" {
		return errors.New(`usage: exascale infer chat -m MODEL "prompt"`)
	}
	if err := requireToken(cfg); err != nil {
		return err
	}
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	model := fs.String("m", "llama-3.1-8b", "model id")
	_ = fs.Parse(args[1:])
	text := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if text == "" {
		return errors.New(`provide a prompt: exascale infer chat -m llama-3.1-8b "hello"`)
	}
	body := map[string]any{"model": *model, "messages": []map[string]string{{"role": "user", "content": text}}}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := client.Do("POST", cfg.GatewayURL, "/v1/chat/completions", cfg.Token, nil, body, &out); err != nil {
		return err
	}
	if len(out.Choices) > 0 {
		fmt.Println(out.Choices[0].Message.Content)
	}
	fmt.Fprintf(os.Stderr, "\n[%d in · %d out · %d tokens]\n", out.Usage.PromptTokens, out.Usage.CompletionTokens, out.Usage.TotalTokens)
	return nil
}

// cmdKeys dispatches the API-key subcommands.
func cmdKeys(cfg config.Config, args []string) error {
	if err := requireToken(cfg); err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("usage: exascale keys create|list|revoke")
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("create", flag.ExitOnError)
		name := fs.String("name", "", "key name")
		_ = fs.Parse(args[1:])
		if *name == "" {
			return errors.New("usage: exascale keys create --name production")
		}
		var out map[string]any
		if err := client.Do("POST", cfg.PlatformURL, "/v1/auth/keys", cfg.Token, nil, map[string]any{"name": *name, "scopes": []string{"inference:read"}}, &out); err != nil {
			return err
		}
		fmt.Printf("Key created — copy it now, it won't be shown again:\n\n  %v\n", out["secret"])
		return nil
	case "list":
		var out struct {
			Keys []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Prefix string `json:"prefix"`
			} `json:"keys"`
		}
		if err := client.Do("GET", cfg.PlatformURL, "/v1/auth/keys", cfg.Token, nil, nil, &out); err != nil {
			return err
		}
		fmt.Printf("%-38s %-16s %s\n", "ID", "PREFIX", "NAME")
		for _, k := range out.Keys {
			fmt.Printf("%-38s %-16s %s\n", k.ID, k.Prefix, k.Name)
		}
		return nil
	case "revoke":
		if len(args) < 2 {
			return errors.New("usage: exascale keys revoke <id>")
		}
		if err := client.Do("DELETE", cfg.PlatformURL, "/v1/auth/keys/"+args[1], cfg.Token, nil, nil, nil); err != nil {
			return err
		}
		fmt.Println("Revoked.")
		return nil
	default:
		return fmt.Errorf("unknown keys subcommand %q", args[0])
	}
}

// cmdConfig views or sets a config value.
func cmdConfig(cfg config.Config, args []string) error {
	if len(args) == 0 || args[0] == "get" {
		fmt.Printf("platform_url  %s\ngateway_url   %s\nledger_url    %s\nlogged_in     %v\nconfig file   %s\n",
			cfg.PlatformURL, cfg.GatewayURL, cfg.LedgerURL, cfg.Token != "", config.Path())
		return nil
	}
	if args[0] == "set" {
		if len(args) < 3 {
			return errors.New("usage: exascale config set <platform_url|gateway_url|ledger_url> <value>")
		}
		switch args[1] {
		case "platform_url":
			cfg.PlatformURL = args[2]
		case "gateway_url":
			cfg.GatewayURL = args[2]
		case "ledger_url":
			cfg.LedgerURL = args[2]
		default:
			return fmt.Errorf("unknown key %q", args[1])
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("Saved.")
		return nil
	}
	return errors.New("usage: exascale config get|set")
}
