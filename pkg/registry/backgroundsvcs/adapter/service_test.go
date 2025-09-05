package adapter

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/stretchr/testify/require"
)

const (
	testContextTimeout  = 100 * time.Millisecond
	expectedMinDuration = 90 * time.Millisecond
)

func TestAsNamedService(t *testing.T) {
	t.Run("creates service adapter with correct properties", func(t *testing.T) {
		mockSvc := &mockService{}
		adapter := asNamedService(mockSvc)

		require.NotNil(t, adapter)
		require.NotNil(t, adapter.BasicService)
		require.Equal(t, mockSvc, adapter.service)

		expectedName := reflect.TypeOf(mockSvc).String()
		require.Equal(t, expectedName, adapter.name)
		require.Equal(t, expectedName, adapter.ServiceName())
	})

	t.Run("implements NamedService interface", func(t *testing.T) {
		mockSvc := &mockService{}
		adapter := asNamedService(mockSvc)

		require.Implements(t, (*services.NamedService)(nil), adapter)
		require.NotEmpty(t, adapter.ServiceName())
		require.Equal(t, services.New, adapter.State())
	})

	t.Run("different service types get different names", func(t *testing.T) {
		mockSvc1 := &mockService{}
		adapter1 := asNamedService(mockSvc1)

		type anotherMockService struct{ mockService }
		mockSvc2 := &anotherMockService{}
		adapter2 := asNamedService(mockSvc2)

		require.Contains(t, adapter1.ServiceName(), "mockService")
		require.Contains(t, adapter2.ServiceName(), "anotherMockService")
		require.NotEqual(t, adapter1.ServiceName(), adapter2.ServiceName())
	})
}

func TestServiceAdapter_Run(t *testing.T) {
	t.Run("delegates to underlying service", func(t *testing.T) {
		mockSvc := &mockService{}
		mockSvc.runFunc = func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		}

		adapter := asNamedService(mockSvc)

		ctx, cancel := context.WithTimeout(context.Background(), testContextTimeout)
		defer cancel()

		err := adapter.run(ctx)
		require.NoError(t, err)
		require.True(t, mockSvc.runCalled)
	})

	t.Run("propagates service errors except context.Canceled", func(t *testing.T) {
		testCases := []struct {
			name        string
			err         error
			expectError bool
		}{
			{"generic error", errors.New("service error"), true},
			{"context canceled", context.Canceled, true},
			{"deadline exceeded", context.DeadlineExceeded, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockSvc := &mockService{runError: tc.err}
				adapter := asNamedService(mockSvc)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				err := adapter.run(ctx)
				if tc.expectError {
					require.Error(t, err)
					require.Equal(t, tc.err, err)
				} else {
					require.NoError(t, err)
				}
				require.True(t, mockSvc.runCalled)
			})
		}
	})

	t.Run("waits for context when service completes successfully", func(t *testing.T) {
		mockSvc := &mockService{} // Service completes immediately without error
		adapter := asNamedService(mockSvc)

		ctx, cancel := context.WithTimeout(context.Background(), testContextTimeout)
		defer cancel()

		start := time.Now()
		err := adapter.run(ctx)
		duration := time.Since(start)

		require.NoError(t, err)
		require.GreaterOrEqual(t, duration, expectedMinDuration)
		require.True(t, mockSvc.runCalled)
	})

	t.Run("handles immediately cancelled context", func(t *testing.T) {
		mockSvc := &mockService{}
		adapter := asNamedService(mockSvc)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := adapter.run(ctx)
		require.NoError(t, err)
		require.True(t, mockSvc.runCalled)
	})

	t.Run("handles context.Canceled from service", func(t *testing.T) {
		mockSvc := &mockService{runError: context.Canceled}
		adapter := asNamedService(mockSvc)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := adapter.run(ctx)
		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
		require.True(t, mockSvc.runCalled)
	})
}
