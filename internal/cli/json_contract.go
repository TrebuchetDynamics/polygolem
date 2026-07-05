package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/output"
	"github.com/spf13/cobra"
)

const commandGroupAnnotation = "polygolem.command_group"

type commandStartedAtKey struct{}

var ErrExit = errors.New("polygolem command exit")

type ExitError struct {
	Err      error
	Code     int
	Rendered bool
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return ErrExit.Error()
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	if e.Err == nil {
		return ErrExit
	}
	return e.Err
}

func (e *ExitError) Is(target error) bool {
	return target == ErrExit || errors.Is(e.Err, target)
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) && exitErr.Code > 0 {
		return exitErr.Code
	}
	return 1
}

func ErrorAlreadyRendered(err error) bool {
	var exitErr *ExitError
	return errors.As(err, &exitErr) && exitErr.Rendered
}

func installJSONContract(root *cobra.Command) {
	for _, cmd := range allCommands(root) {
		wrapArgs(cmd)
		wrapRunE(cmd)
	}
}

func allCommands(root *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		out = append(out, cmd)
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(root)
	return out
}

func wrapRunE(cmd *cobra.Command) {
	if cmd.RunE == nil {
		return
	}
	orig := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		startedAt := time.Now()
		cmd.SetContext(context.WithValue(cmd.Context(), commandStartedAtKey{}, startedAt))
		if jsonEnabled(cmd) && cmd.Annotations[commandGroupAnnotation] == "true" {
			return renderCommandError(cmd, newUsageSubcommandError(cmd))
		}
		err := orig(cmd, args)
		if err == nil {
			return nil
		}
		if !jsonEnabled(cmd) {
			return &ExitError{Err: err, Code: exitCodeForOutputError(classifyCommandError(err)), Rendered: false}
		}
		return renderCommandError(cmd, err)
	}
}

func wrapArgs(cmd *cobra.Command) {
	if cmd.Args == nil {
		return
	}
	orig := cmd.Args
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if _, ok := cmd.Context().Value(commandStartedAtKey{}).(time.Time); !ok {
			cmd.SetContext(context.WithValue(cmd.Context(), commandStartedAtKey{}, time.Now()))
		}
		err := orig(cmd, args)
		if err == nil {
			return nil
		}
		if !jsonEnabled(cmd) {
			return &ExitError{Err: err, Code: exitCodeForOutputError(classifyCommandError(err)), Rendered: false}
		}
		return renderCommandError(cmd, err)
	}
}

func writeCommandJSON(cmd *cobra.Command, v any) error {
	if jsonEnabled(cmd) {
		return output.WriteSuccess(cmd.OutOrStdout(), commandName(cmd), commandStartedAt(cmd), v)
	}
	return output.WriteJSON(cmd.OutOrStdout(), v)
}

func renderCommandError(cmd *cobra.Command, err error) error {
	if exitErr, ok := err.(*ExitError); ok {
		return exitErr
	}
	outErr := classifyCommandError(err)
	code := exitCodeForOutputError(outErr)
	if jsonEnabled(cmd) {
		_ = output.WriteErrorEnvelope(cmd.ErrOrStderr(), commandName(cmd), commandStartedAt(cmd), outErr)
		return &ExitError{Err: err, Code: code, Rendered: true}
	}
	return &ExitError{Err: err, Code: code, Rendered: false}
}

func jsonEnabled(cmd *cobra.Command) bool {
	flag := cmd.Root().PersistentFlags().Lookup("json")
	if flag == nil {
		return false
	}
	v, err := strconv.ParseBool(flag.Value.String())
	return err == nil && v
}

func commandStartedAt(cmd *cobra.Command) time.Time {
	startedAt, _ := cmd.Context().Value(commandStartedAtKey{}).(time.Time)
	return startedAt
}

func commandName(cmd *cobra.Command) string {
	path := cmd.CommandPath()
	rootPath := cmd.Root().CommandPath()
	if path == rootPath {
		return rootPath
	}
	return strings.TrimPrefix(path, rootPath+" ")
}

type usageSubcommandError struct {
	command string
}

func newUsageSubcommandError(cmd *cobra.Command) error {
	return usageSubcommandError{command: commandName(cmd)}
}

func (e usageSubcommandError) Error() string {
	return fmt.Sprintf("%s requires a subcommand", e.command)
}

func classifyCommandError(err error) output.Error {
	var groupErr usageSubcommandError
	if errors.As(err, &groupErr) {
		return output.Error{
			Code:     "USAGE_SUBCOMMAND_UNKNOWN",
			Category: "usage",
			Message:  groupErr.Error(),
			Hint:     "Run the command with --help to list available subcommands.",
		}
	}

	msg := err.Error()
	if isNetworkCommandError(msg) {
		return output.Error{
			Code:     "NETWORK_UPSTREAM_UNREACHABLE",
			Category: "network",
			Message:  msg,
			Hint:     "Retry after checking network connectivity and configured Polymarket endpoints.",
			Details:  map[string]string{"source": "transport"},
		}
	}
	if isChainCommandError(msg) {
		return output.Error{
			Code:     "CHAIN_RPC_ERROR",
			Category: "chain",
			Message:  msg,
			Hint:     "Check Polygon RPC status, wallet balances, allowances, nonce, and transaction preconditions.",
			Details:  map[string]string{"source": "polygon_rpc"},
		}
	}
	if httpStatus, ok := extractHTTPStatus(msg); ok {
		return output.Error{
			Code:     "PROTOCOL_UPSTREAM_HTTP_ERROR",
			Category: "protocol",
			Message:  msg,
			Hint:     "Inspect error.details.upstream_status and verify the upstream CLOB/relayer/API preconditions.",
			Details: map[string]string{
				"source":          upstreamSourceFromMessage(msg),
				"upstream_status": httpStatus,
			},
		}
	}

	switch {
	case strings.Contains(msg, "SIGNER_PRIVATE_KEY is required") || strings.Contains(msg, "POLYMARKET_PRIVATE_KEY is required"):
		return output.Error{
			Code:     "AUTH_PRIVATE_KEY_MISSING",
			Category: "auth",
			Message:  "SIGNER_PRIVATE_KEY is required.",
			Hint:     "Set SIGNER_PRIVATE_KEY in the environment before running authenticated commands.",
		}
	case strings.Contains(msg, "builder credentials not configured"):
		return output.Error{
			Code:     "AUTH_BUILDER_MISSING",
			Category: "auth",
			Message:  msg,
			Hint:     "Run auth login or configure relayer credentials before deposit-wallet commands.",
		}
	case strings.Contains(msg, "not implemented"):
		return output.Error{
			Code:     "INTERNAL_UNIMPLEMENTED",
			Category: "internal",
			Message:  msg,
		}
	case strings.HasPrefix(msg, "--") && strings.Contains(msg, "required"):
		return output.Error{
			Code:     "USAGE_FLAG_MISSING",
			Category: "usage",
			Message:  msg,
		}
	case strings.Contains(msg, "arg(s)") || strings.Contains(msg, "accepts ") || strings.Contains(msg, "requires "):
		return output.Error{
			Code:     "USAGE_ARG_INVALID",
			Category: "usage",
			Message:  msg,
		}
	case strings.Contains(msg, "only --output json is supported"):
		return output.Error{
			Code:     "USAGE_FLAG_INVALID",
			Category: "usage",
			Message:  msg,
		}
	default:
		return output.Error{
			Code:     "INTERNAL_COMMAND_FAILED",
			Category: "internal",
			Message:  msg,
		}
	}
}

func isNetworkCommandError(msg string) bool {
	lower := strings.ToLower(msg)
	for _, needle := range []string{
		"context deadline exceeded",
		"client.timeout",
		"connection refused",
		"no such host",
		"network is unreachable",
		"tls handshake timeout",
		"i/o timeout",
	} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func isChainCommandError(msg string) bool {
	lower := strings.ToLower(msg)
	for _, needle := range []string{
		"execution reverted",
		"insufficient funds",
		"nonce too low",
		"replacement transaction underpriced",
		"transaction underpriced",
		"eth_call",
		"eth_sendrawtransaction",
		"erc20 allowance",
	} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func extractHTTPStatus(msg string) (string, bool) {
	fields := strings.FieldsFunc(msg, func(r rune) bool {
		return r == ' ' || r == ':' || r == '(' || r == ')' || r == ',' || r == ';'
	})
	for i, field := range fields {
		if strings.EqualFold(field, "HTTP") && i+1 < len(fields) {
			status := strings.TrimSpace(fields[i+1])
			if len(status) == 3 {
				if _, err := strconv.Atoi(status); err == nil {
					return status, true
				}
			}
		}
		if strings.HasPrefix(strings.ToLower(field), "http") && len(field) >= 7 {
			status := field[len(field)-3:]
			if _, err := strconv.Atoi(status); err == nil {
				return status, true
			}
		}
	}
	return "", false
}

func upstreamSourceFromMessage(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "relayer"):
		return "relayer"
	case strings.Contains(lower, "clob"):
		return "clob"
	case strings.Contains(lower, "gamma"):
		return "gamma"
	case strings.Contains(lower, "data"):
		return "data_api"
	default:
		return "upstream_http"
	}
}

func exitCodeForOutputError(err output.Error) int {
	switch err.Category {
	case "usage":
		return 2
	case "auth":
		return 3
	case "validation":
		return 4
	case "gate":
		return 5
	case "network":
		return 6
	case "protocol":
		return 7
	case "chain":
		return 8
	case "internal":
		return 9
	default:
		return 1
	}
}
