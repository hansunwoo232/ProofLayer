package executor

import "context"

type ScenarioHandler interface {
	Execute(context.Context, string) error
	Cleanup(context.Context, string) error
	VerifyAbsent(context.Context, string) error
}

type handlerFunctions struct {
	execute      func(context.Context, string) error
	cleanup      func(context.Context, string) error
	verifyAbsent func(context.Context, string) error
}

func (handler handlerFunctions) Execute(ctx context.Context, correlationID string) error {
	return handler.execute(ctx, correlationID)
}

func (handler handlerFunctions) Cleanup(ctx context.Context, correlationID string) error {
	return handler.cleanup(ctx, correlationID)
}

func (handler handlerFunctions) VerifyAbsent(ctx context.Context, correlationID string) error {
	return handler.verifyAbsent(ctx, correlationID)
}

func noCleanup(ctx context.Context, _ string) error {
	return ctx.Err()
}

func builtInHandlers() map[string]ScenarioHandler {
	return map[string]ScenarioHandler{
		"builtin.emit_process_marker":          processMarkerHandler(),
		"builtin.create_registry_canary":       registryCanaryHandler(),
		"builtin.create_scheduled_task_canary": scheduledTaskCanaryHandler(),
	}
}
