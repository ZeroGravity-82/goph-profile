package readiness

import (
	"context"
	"errors"
	"time"
)

// DefaultTimeout ограничивает время выполнения каждой проверки готовности.
const DefaultTimeout = 2 * time.Second

// Check проверяет готовность внешней зависимости.
type Check func(ctx context.Context) error

// Checks содержит именованные проверки готовности внешних зависимостей.
type Checks map[string]Check

// Results содержит результаты проверок готовности внешних зависимостей.
type Results map[string]error

type checkResult struct {
	name string
	err  error
}

// Run параллельно выполняет проверки готовности с отдельным тайм-аутом для каждой проверки.
func Run(ctx context.Context, checks Checks, timeout time.Duration) Results {
	results := make(Results, len(checks))
	resultCh := make(chan checkResult, len(checks))

	for name, check := range checks {
		go func(name string, check Check) {
			checkCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			resultCh <- checkResult{name: name, err: check(checkCtx)}
		}(name, check)
	}

	for range checks {
		result := <-resultCh
		results[result.name] = result.err
	}

	return results
}

// Err объединяет ошибки всех неуспешных проверок.
func (r Results) Err() error {
	var result error
	for _, err := range r {
		result = errors.Join(result, err)
	}
	return result
}
