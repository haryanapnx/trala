package usecase

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/haryanapnx/trala/internal/model"
)

// TestTaskCreateConcurrentDuplicateKeyCreatesExactlyOneTask fires many
// simultaneous requests sharing one Idempotency-Key and asserts the
// exactly-once guarantee: one task, identical responses. Run with -race.
func TestTaskCreateConcurrentDuplicateKeyCreatesExactlyOneTask(t *testing.T) {
	tasks := newFakeTaskRepo()
	statuses := &fakeStatusRepo{inUse: map[string]bool{}}
	teams := newFakeTeamRepo()
	idempotency := newFakeIdempotencyRepo()
	txm := &txManager{tasks: tasks, idempotency: idempotency}
	uc := NewTaskUsecase(tasks, statuses, teams, idempotency, &fakeTaskLogRepo{}, &fakeNotifier{}, txm, fakeTaskEncoder)

	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")

	const goroutines = 25
	in := model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "race", IdempotencyKey: "shared-key"}

	var wg sync.WaitGroup
	bodies := make([][]byte, goroutines)
	errs := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := uc.Create(context.Background(), in)
			errs[i] = err
			if res != nil {
				bodies[i] = res.Body
			}
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "goroutine %d failed", i)
	}

	assert.Len(t, tasks.tasks, 1, "exactly one task must exist for one idempotency key")
	for i := 1; i < goroutines; i++ {
		assert.Equal(t, bodies[0], bodies[i], "every response must be identical")
	}
}

// TestTaskCreateConcurrentDistinctKeysCreatesAll is the sanity counterpart: with
// distinct keys, every concurrent request creates its own task.
func TestTaskCreateConcurrentDistinctKeysCreatesAll(t *testing.T) {
	tasks := newFakeTaskRepo()
	statuses := &fakeStatusRepo{inUse: map[string]bool{}}
	teams := newFakeTeamRepo()
	idempotency := newFakeIdempotencyRepo()
	txm := &txManager{tasks: tasks, idempotency: idempotency}
	uc := NewTaskUsecase(tasks, statuses, teams, idempotency, &fakeTaskLogRepo{}, &fakeNotifier{}, txm, fakeTaskEncoder)

	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")

	const goroutines = 10

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := uc.Create(context.Background(), model.CreateTaskInput{
				UserID: "u1", TeamID: "t1", Title: "unique " + strconv.Itoa(i),
				IdempotencyKey: "key-" + strconv.Itoa(i),
			})
			require.NoError(t, err)
		}(i)
	}
	wg.Wait()

	assert.Len(t, tasks.tasks, goroutines)
}
