package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/exascale/cli/internal/client"
	"github.com/exascale/cli/internal/config"
	"github.com/exascale/cli/internal/ui"
)

// modelMeta is the per-model billing info the REPL needs for its live cost meter.
type modelMeta struct {
	price      float64 // credits per 1K tokens
	creditType string
}

// cmdChat runs a persistent, multi-turn, streaming chat session (F24 phase 2) — the headline
// conversational surface. The prompt shows live context (model · balance), slash commands manage the
// session, and each turn streams the reply then meters its cost against a locally-tracked balance.
func cmdChat(cfg config.Config, args []string) error {
	if err := requireToken(cfg); err != nil {
		return err
	}
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	model := fs.String("m", "llama-3.1-8b", "model id to start with")
	_ = fs.Parse(args)

	meta := fetchModelMeta(cfg) // id → {price, creditType}
	creditType := creditTypeOf(meta, *model)
	balance := fetchBalance(cfg, creditType) // tracked locally; reconciled by /balance
	var history []map[string]string
	var sessionCost float64
	in := bufio.NewReader(os.Stdin)

	fmt.Fprintln(os.Stderr, ui.Dim("exascale chat — type a message; /help for commands, /exit to quit"))
	for {
		fmt.Printf("%s %s %s ", ui.Cyan(*model), ui.Dim("· "+ui.Num(balance)+" "+creditType), ui.Bold("▸"))
		line, err := in.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) { // Ctrl-D — clean exit
				fmt.Println()
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			cmd, rest := splitSlash(line)
			switch cmd {
			case "exit", "quit", "q":
				return nil
			case "help", "?":
				printChatHelp()
			case "model":
				if rest == "" {
					fmt.Fprintln(os.Stderr, ui.Dim("current model: "+*model))
				} else {
					*model = rest
					creditType = creditTypeOf(meta, *model)
					balance = fetchBalance(cfg, creditType)
					fmt.Fprintln(os.Stderr, ui.Dim("switched to "+*model))
				}
			case "balance":
				balance = fetchBalance(cfg, creditType)
				fmt.Fprintln(os.Stderr, ui.Dim(ui.Num(balance)+" "+creditType+" available"))
			case "cost":
				fmt.Fprintln(os.Stderr, ui.Dim(fmt.Sprintf("session cost: %s %s over %d turn(s)", ui.Num(sessionCost), creditType, len(history)/2)))
			case "clear":
				history = nil
				fmt.Fprintln(os.Stderr, ui.Dim("conversation cleared"))
			case "save":
				if err := saveTranscript(rest, history); err != nil {
					fmt.Fprintln(os.Stderr, ui.Red("save failed: ")+err.Error())
				} else {
					fmt.Fprintln(os.Stderr, ui.Dim("saved transcript"))
				}
			default:
				fmt.Fprintln(os.Stderr, ui.Red("unknown command")+" /"+cmd+" — try /help")
			}
			continue
		}

		// A chat turn: append the user message, stream the reply over the full history.
		history = append(history, map[string]string{"role": "user", "content": line})
		var reply strings.Builder
		body := map[string]any{"model": *model, "messages": history, "stream": true}
		usage, err := client.StreamChat(cfg.GatewayURL, "/v1/chat/completions", cfg.Token, body, func(tok string) {
			reply.WriteString(tok)
			fmt.Print(tok)
		})
		fmt.Println()
		if err != nil {
			history = history[:len(history)-1] // drop the failed turn so history stays valid
			fmt.Fprintln(os.Stderr, ui.Red("error: ")+err.Error())
			var ae *client.APIError
			if errors.As(err, &ae) {
				if h := ui.Hint(ae.Status, ae.Code); h != "" {
					fmt.Fprintln(os.Stderr, ui.Dim("  → "+h))
				}
			}
			continue
		}
		history = append(history, map[string]string{"role": "assistant", "content": reply.String()})

		cost := meta[*model].price * float64(usage.TotalTokens) / 1000.0
		sessionCost += cost
		balance -= cost
		if usage.TotalTokens > 0 {
			fmt.Fprintln(os.Stderr, ui.Dim(fmt.Sprintf("%s %s · %s left · %d tok", ui.Delta(-cost), creditType, ui.Num(balance), usage.TotalTokens)))
		}
	}
}

// parseFloat parses a fixed-point credit/price string to a float for display + the cost meter; a bad
// value yields 0 (cosmetic only — the ledger remains the source of truth for actual balances).
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

// splitSlash parses a "/cmd rest of line" into the lowercase command and its trailing argument.
func splitSlash(line string) (cmd, rest string) {
	line = strings.TrimPrefix(strings.TrimSpace(line), "/")
	c, r, _ := strings.Cut(line, " ")
	return strings.ToLower(c), strings.TrimSpace(r)
}

// printChatHelp lists the REPL slash commands.
func printChatHelp() {
	fmt.Fprintln(os.Stderr, strings.TrimSpace(`
  /model [id]   show or switch the model        /balance   refresh + show balance
  /cost         session cost so far             /clear     reset the conversation
  /save [file]  write the transcript to a file  /exit      leave the chat`))
}

// saveTranscript writes the conversation as plain "role: content" lines (default exascale-chat.txt).
func saveTranscript(path string, history []map[string]string) error {
	if path == "" {
		path = "exascale-chat.txt"
	}
	var b strings.Builder
	for _, m := range history {
		b.WriteString(strings.ToUpper(m["role"]))
		b.WriteString(": ")
		b.WriteString(m["content"])
		b.WriteString("\n\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

// fetchModelMeta loads the catalog once into an id→{price,creditType} map for the cost meter. A failure
// degrades to an empty map (cost shows zero), never blocking the chat.
func fetchModelMeta(cfg config.Config) map[string]modelMeta {
	out := map[string]modelMeta{}
	var resp struct {
		Data []struct {
			ID       string `json:"id"`
			Exascale struct {
				CreditType string `json:"credit_type"`
				Price      string `json:"price"`
			} `json:"exascale"`
		} `json:"data"`
	}
	if client.Do("GET", cfg.GatewayURL, "/v1/models", cfg.Token, nil, nil, &resp) != nil {
		return out
	}
	for _, m := range resp.Data {
		out[m.ID] = modelMeta{price: parseFloat(m.Exascale.Price), creditType: m.Exascale.CreditType}
	}
	return out
}

// creditTypeOf returns the model's billing credit type, defaulting to "text" when unknown.
func creditTypeOf(meta map[string]modelMeta, id string) string {
	if m, ok := meta[id]; ok && m.creditType != "" {
		return m.creditType
	}
	return "text"
}

// fetchBalance returns the available balance for one credit type (0 when absent/unreachable).
func fetchBalance(cfg config.Config, creditType string) float64 {
	var resp struct {
		Balances []struct {
			CreditType string `json:"credit_type"`
			Balance    string `json:"balance"`
		} `json:"balances"`
	}
	if client.Do("GET", cfg.LedgerURL, "/v1/credits/balances", cfg.Token, nil, nil, &resp) != nil {
		return 0
	}
	for _, b := range resp.Balances {
		if b.CreditType == creditType {
			return parseFloat(b.Balance)
		}
	}
	return 0
}
