package saga_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/saga"
)

func TestSaga_Execute_AllStepsSucceed(t *testing.T) {
	t.Parallel()

	executionOrder := []string{}

	s := saga.New("TestSaga", nil)
	s.AddStep("Step1",
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step1-forward")
			return nil
		},
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step1-compensate")
			return nil
		},
	)
	s.AddStep("Step2",
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step2-forward")
			return nil
		},
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step2-compensate")
			return nil
		},
	)
	s.AddStep("Step3",
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step3-forward")
			return nil
		},
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step3-compensate")
			return nil
		},
	)

	err := s.Execute(context.Background())

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedOrder := []string{"step1-forward", "step2-forward", "step3-forward"}
	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("Expected %d executions, got %d", len(expectedOrder), len(executionOrder))
	}
	for i, expected := range expectedOrder {
		if executionOrder[i] != expected {
			t.Errorf("Expected %s at position %d, got %s", expected, i, executionOrder[i])
		}
	}
}

func TestSaga_Execute_SecondStepFails_RollsBackFirst(t *testing.T) {
	t.Parallel()

	executionOrder := []string{}
	step2Error := errors.New("step2 failed")

	s := saga.New("TestSaga", nil)
	s.AddStep("Step1",
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step1-forward")
			return nil
		},
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step1-compensate")
			return nil
		},
	)
	s.AddStep("Step2",
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step2-forward")
			return step2Error
		},
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step2-compensate")
			return nil
		},
	)
	s.AddStep("Step3",
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step3-forward")
			return nil
		},
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step3-compensate")
			return nil
		},
	)

	err := s.Execute(context.Background())

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !errors.Is(err, step2Error) {
		t.Errorf("Expected error to wrap step2Error, got: %v", err)
	}

	// Step3 should NOT be executed
	expectedOrder := []string{"step1-forward", "step2-forward", "step1-compensate"}
	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("Expected %d executions, got %d: %v", len(expectedOrder), len(executionOrder), executionOrder)
	}
	for i, expected := range expectedOrder {
		if i >= len(executionOrder) {
			t.Errorf("Missing execution at position %d, expected %s", i, expected)
			continue
		}
		if executionOrder[i] != expected {
			t.Errorf("Expected %s at position %d, got %s", expected, i, executionOrder[i])
		}
	}
}

func TestSaga_Execute_ThirdStepFails_RollsBackFirstTwo(t *testing.T) {
	t.Parallel()

	executionOrder := []string{}

	s := saga.New("TestSaga", nil)
	s.AddStep("Step1",
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step1-forward")
			return nil
		},
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step1-compensate")
			return nil
		},
	)
	s.AddStep("Step2",
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step2-forward")
			return nil
		},
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step2-compensate")
			return nil
		},
	)
	s.AddStep("Step3",
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step3-forward")
			return errors.New("step3 failed")
		},
		func(ctx context.Context) error {
			executionOrder = append(executionOrder, "step3-compensate")
			return nil
		},
	)

	err := s.Execute(context.Background())

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Rollback should happen in reverse order
	expectedOrder := []string{
		"step1-forward",
		"step2-forward",
		"step3-forward",
		"step2-compensate",
		"step1-compensate",
	}
	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("Expected %d executions, got %d: %v", len(expectedOrder), len(executionOrder), executionOrder)
	}
	for i, expected := range expectedOrder {
		if i >= len(executionOrder) {
			t.Errorf("Missing execution at position %d, expected %s", i, expected)
			continue
		}
		if executionOrder[i] != expected {
			t.Errorf("Expected %s at position %d, got %s", expected, i, executionOrder[i])
		}
	}
}

func TestSaga_Execute_EmptySaga_Succeeds(t *testing.T) {
	t.Parallel()

	s := saga.New("EmptySaga", nil)

	err := s.Execute(context.Background())

	if err != nil {
		t.Errorf("Expected no error for empty saga, got: %v", err)
	}
}

func TestSaga_ExecuteWithResult_Success(t *testing.T) {
	t.Parallel()

	s := saga.New("TestSaga", nil)
	s.AddStep("Step1",
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return nil },
	)

	result := s.ExecuteWithResult(context.Background())

	if !result.Success {
		t.Error("Expected success=true")
	}
	if result.FailedStep != "" {
		t.Errorf("Expected empty FailedStep, got: %s", result.FailedStep)
	}
	if result.Error != nil {
		t.Errorf("Expected nil error, got: %v", result.Error)
	}
}

func TestSaga_ExecuteWithResult_Failure(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("step failed")

	s := saga.New("TestSaga", nil)
	s.AddStep("FailingStep",
		func(ctx context.Context) error { return expectedError },
		func(ctx context.Context) error { return nil },
	)

	result := s.ExecuteWithResult(context.Background())

	if result.Success {
		t.Error("Expected success=false")
	}
	if result.FailedStep != "FailingStep" {
		t.Errorf("Expected FailedStep='FailingStep', got: %s", result.FailedStep)
	}
	if !errors.Is(result.Error, expectedError) {
		t.Errorf("Expected error to be %v, got: %v", expectedError, result.Error)
	}
}

func TestSaga_CompensationFailure_ReturnsError(t *testing.T) {
	t.Parallel()

	compensationError := errors.New("compensation failed")

	s := saga.New("TestSaga", nil)
	s.AddStep("Step1",
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return compensationError },
	)
	s.AddStep("Step2",
		func(ctx context.Context) error { return errors.New("step2 failed") },
		func(ctx context.Context) error { return nil },
	)

	result := s.ExecuteWithResult(context.Background())

	if result.Success {
		t.Error("Expected success=false")
	}
	if result.RollbackError == nil {
		t.Error("Expected RollbackError to be set")
	}
}

func TestSaga_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	s := saga.New("TestSaga", nil)
	s.AddStep("Step1",
		func(c context.Context) error {
			return c.Err() // Check context
		},
		func(c context.Context) error { return nil },
	)

	err := s.Execute(ctx)

	if err == nil {
		t.Error("Expected error due to cancelled context")
	}
}
