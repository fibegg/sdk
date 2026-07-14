package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func withRawInputSchema(schema map[string]any) mcp.ToolOption {
	return func(t *mcp.Tool) {
		raw, err := json.Marshal(schema)
		if err != nil {
			return
		}
		t.InputSchema.Type = ""
		t.RawInputSchema = json.RawMessage(raw)
	}
}

type toolOpts struct {
	Tier             toolTier
	Idempotent       bool
	Hidden           bool
	ExtraSchemaProps []mcp.ToolOption
}

// addTool registers implementations before applying advertisement filters;
// pipelines may call tools hidden from the current public tier.
func (s *Server) addTool(t *toolImpl, tool mcp.Tool) {
	s.dispatcher.register(t)

	// mcp-go defaults destructiveHint to true, so set every hint explicitly.
	mcp.WithReadOnlyHintAnnotation(t.annotations.ReadOnly)(&tool)
	mcp.WithDestructiveHintAnnotation(t.annotations.Destructive)(&tool)
	mcp.WithIdempotentHintAnnotation(t.annotations.Idempotent)(&tool)
	enrichToolInputSchema(t.name, &tool)
	makeToolInputSchemaClientCompatible(&tool)
	if schema, ok := toolInputSchemaToMap(tool).(map[string]any); ok {
		s.toolSchemas[t.name] = schema
	}
	if !s.includeTool(t) {
		return
	}

	s.mcp.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		if args == nil {
			args = map[string]any{}
		}

		start := time.Now()
		result, err := s.dispatcher.dispatch(ctx, t.name, args)
		s.auditLog(ctx, t.name, args, err, time.Since(start))
		if err != nil {
			return toolResultFromError(t.name, err), nil
		}
		if result == nil {
			return mcp.NewToolResultText("{}"), nil
		}
		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	})
}

// toolResultFromError preserves structured fields so MCP clients can classify
// failures without parsing human-readable messages.
func toolResultFromError(toolName string, err error) *mcp.CallToolResult {
	payload := map[string]any{
		"tool":    toolName,
		"message": err.Error(),
	}

	var apiErr *fibe.APIError
	if errors.As(err, &apiErr) {
		payload["code"] = apiErr.Code
		payload["status"] = apiErr.StatusCode
		payload["message"] = apiErr.Message
		if apiErr.RequestID != "" {
			payload["request_id"] = apiErr.RequestID
		}
		if apiErr.Details != nil {
			payload["details"] = apiErr.Details
		}
		if apiErr.IdempotentReplayed {
			payload["idempotent_replayed"] = true
		}
		if apiErr.RetryAfter > 0 {
			payload["retry_after_seconds"] = apiErr.RetryAfter.Seconds()
		}
	} else if cbErr, ok := err.(*fibe.CircuitOpenError); ok {
		payload["code"] = "CIRCUIT_OPEN"
		payload["resource"] = cbErr.Resource
	} else if _, ok := err.(*confirmRequiredError); ok {
		payload["code"] = "CONFIRM_REQUIRED"
		payload["hint"] = "pass confirm:true or run server with --yolo"
	} else if code, status, details, requestID := structuredToolErrorFields(err); code != "UNKNOWN_ERROR" || status != 500 || details != nil || requestID != "" {
		payload["code"] = code
		payload["status"] = status
		if details != nil {
			payload["details"] = details
		}
		if requestID != "" {
			payload["request_id"] = requestID
		}
	}

	body, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return mcp.NewToolResultError(err.Error())
	}
	return mcp.NewToolResultError(string(body))
}

type structuredToolError interface {
	error
	ErrorCode() string
	ErrorStatus() int
	ErrorDetails() map[string]any
}

func structuredToolErrorFields(err error) (string, int, map[string]any, string) {
	var apiErr *fibe.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code, apiErr.StatusCode, apiErr.Details, apiErr.RequestID
	}

	var terminalErr *fibe.PlaygroundTerminalError
	if errors.As(err, &terminalErr) {
		return fibe.ErrCodePlaygroundTerminalState, 422, terminalErr.Details(), ""
	}

	var structured structuredToolError
	if errors.As(err, &structured) {
		status := structured.ErrorStatus()
		if status == 0 {
			status = 500
		}
		code := structured.ErrorCode()
		if code == "" {
			code = "UNKNOWN_ERROR"
		}
		return code, status, structured.ErrorDetails(), ""
	}

	return "UNKNOWN_ERROR", 500, nil, ""
}

// registerCreate registers the two repository-creation tools that share the
// same request/response shape. Other tools use explicit registration so their
// schemas and safety metadata remain visible at the call site.
func registerCreate[P any, R any](s *Server, name, description string, opts toolOpts,
	fn func(context.Context, *fibe.Client, *P) (*R, error),
) {
	t := &toolImpl{
		name:        name,
		description: description,
		tier:        opts.Tier,
		hidden:      opts.Hidden,
		annotations: toolAnnotations{Idempotent: opts.Idempotent},
		handler: func(ctx context.Context, client *fibe.Client, args map[string]any) (any, error) {
			var params P
			if err := bindArgs(args, &params); err != nil {
				return nil, err
			}
			return fn(ctx, client, &params)
		},
	}

	mcpOpts := []mcp.ToolOption{
		mcp.WithDescription(description),
		mcp.WithInputSchema[P](),
	}
	mcpOpts = append(mcpOpts, opts.ExtraSchemaProps...)
	s.addTool(t, mcp.NewTool(name, mcpOpts...))
}
