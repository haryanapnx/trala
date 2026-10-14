package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/haryfun/task-management-service/internal/domain"
	"github.com/haryfun/task-management-service/internal/model"
)

func newTaskUsecaseFull() (*TaskUsecase, *fakeTaskRepo, *fakeStatusRepo, *fakeTeamRepo, *fakeIdempotencyRepo) {
	tasks := newFakeTaskRepo()
	statuses := &fakeStatusRepo{inUse: map[string]bool{}}
	teams := newFakeTeamRepo()
	idempotency := newFakeIdempotencyRepo()

	uc := NewTaskUsecase(tasks, statuses, teams, idempotency, &fakeTaskLogRepo{}, &fakeNotifier{}, fakeTxManager{}, fakeTaskEncoder)
	return uc, tasks, statuses, teams, idempotency
}

func newTaskUsecase() (*TaskUsecase, *fakeStatusRepo, *fakeTeamRepo) {
	uc, _, statuses, teams, _ := newTaskUsecaseFull()
	return uc, statuses, teams
}

func createTaskForTest(t *testing.T, uc *TaskUsecase, in model.CreateTaskInput) *domain.Task {
	t.Helper()
	res, err := uc.Create(context.Background(), in)
	require.NoError(t, err)
	require.NotNil(t, res.Task)
	return res.Task
}

func TestTaskCreateSuccess(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	todo := mustCreateStatusID(t, statuses, "u1", "todo")

	task := createTaskForTest(t, uc, model.CreateTaskInput{
		UserID: "u1", TeamID: "t1", StatusID: todo, Title: "  Write docs  ",
	})
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "Write docs", task.Title, "title should be trimmed")
	assert.Equal(t, "u1", task.CreatorID)
	assert.Equal(t, "t1", task.TeamID)
	assert.Equal(t, todo, task.StatusID)
	assert.Nil(t, task.AssigneeID)
}

func TestTaskCreateUsesFirstStatusWhenOmitted(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	todo := mustCreateStatusID(t, statuses, "u1", "todo")

	task := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "No status given"})
	assert.Equal(t, todo, task.StatusID, "should default to the first status by position")
}

func TestTaskCreateRejectsEmptyTitle(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")

	_, err := uc.Create(context.Background(), model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "   "})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeValidation, appErr.Code)
}

func TestTaskCreateRejectsNonMember(t *testing.T) {
	uc, statuses, _ := newTaskUsecase()
	seedDefaults(t, statuses, "u1")

	_, err := uc.Create(context.Background(), model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "x"})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeForbidden, appErr.Code)
}

func TestTaskCreateRejectsForeignStatus(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	seedDefaults(t, statuses, "u2")
	teams.addMember("t1", "u1")
	othersStatus := mustCreateStatusID(t, statuses, "u2", "todo")

	_, err := uc.Create(context.Background(), model.CreateTaskInput{
		UserID: "u1", TeamID: "t1", StatusID: othersStatus, Title: "x",
	})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeValidation, appErr.Code)
}

func TestTaskCreateCreatesNewWhenKeyEmpty(t *testing.T) {
	uc, tasks, statuses, teams, _ := newTaskUsecaseFull()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")

	createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "first"})
	createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "second"})

	assert.Len(t, tasks.tasks, 2, "no key means no deduplication")
}

func TestTaskCreateSequentialReplay(t *testing.T) {
	uc, tasks, statuses, teams, _ := newTaskUsecaseFull()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")

	in := model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "once", IdempotencyKey: "key-1"}

	first, err := uc.Create(context.Background(), in)
	require.NoError(t, err)
	require.NotNil(t, first.Task)
	assert.False(t, first.Replayed, "a fresh create is not a replay")

	second, err := uc.Create(context.Background(), in)
	require.NoError(t, err)

	assert.Nil(t, second.Task, "replay must not create a new task")
	assert.True(t, second.Replayed, "the stored response must be marked as replayed")
	assert.Equal(t, first.StatusCode, second.StatusCode)
	assert.Equal(t, first.Body, second.Body, "replayed body must be identical")
	assert.Len(t, tasks.tasks, 1, "only one task may exist for one idempotency key")
}

func TestTaskCreateReuseKeyDifferentPayload(t *testing.T) {
	uc, _, statuses, teams, _ := newTaskUsecaseFull()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")

	_, err := uc.Create(context.Background(), model.CreateTaskInput{
		UserID: "u1", TeamID: "t1", Title: "original", IdempotencyKey: "key-2",
	})
	require.NoError(t, err)

	_, err = uc.Create(context.Background(), model.CreateTaskInput{
		UserID: "u1", TeamID: "t1", Title: "different", IdempotencyKey: "key-2",
	})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeIdempotencyReused, appErr.Code)
}

func TestTaskCreateExpiredKeyTreatedAsNew(t *testing.T) {
	uc, tasks, statuses, teams, idempotency := newTaskUsecaseFull()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")

	idempotency.records["key-3"] = &domain.IdempotencyRecord{
		Key:            "key-3",
		UserID:         "u1",
		RequestHash:    "stale",
		ResponseStatus: 201,
		ResponseBody:   []byte(`{"stale":true}`),
		CreatedAt:      time.Now().Add(-25 * time.Hour),
	}

	res, err := uc.Create(context.Background(), model.CreateTaskInput{
		UserID: "u1", TeamID: "t1", Title: "fresh", IdempotencyKey: "key-3",
	})
	require.NoError(t, err)
	require.NotNil(t, res.Task, "an expired key should create a new task")
	assert.Len(t, tasks.tasks, 1)
}

func TestTaskGetByCreator(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	todo := mustCreateStatusID(t, statuses, "u1", "todo")

	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", StatusID: todo, Title: "T"})

	got, err := uc.Get(context.Background(), "u1", created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestTaskGetByAssignee(t *testing.T) {
	tasks := newFakeTaskRepo()
	uc := NewTaskUsecase(tasks, &fakeStatusRepo{inUse: map[string]bool{}},
		newFakeTeamRepo(), newFakeIdempotencyRepo(), &fakeTaskLogRepo{}, &fakeNotifier{},
		fakeTxManager{}, fakeTaskEncoder)

	assignee := "u2"
	created, err := tasks.Create(context.Background(), &domain.Task{
		TeamID: "t1", CreatorID: "u1", AssigneeID: &assignee, StatusID: "s1", Title: "T",
	})
	require.NoError(t, err)

	got, err := uc.Get(context.Background(), "u2", created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
}

func TestTaskGetByStrangerIsNotFound(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	todo := mustCreateStatusID(t, statuses, "u1", "todo")

	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", StatusID: todo, Title: "T"})

	_, err := uc.Get(context.Background(), "u9", created.ID)
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeNotFound, appErr.Code, "must not leak task existence")
}

func TestTaskGetUnknownIsNotFound(t *testing.T) {
	uc, _, _ := newTaskUsecase()

	_, err := uc.Get(context.Background(), "u1", "does-not-exist")
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeNotFound, appErr.Code)
}

func TestTaskListPaginationAndMeta(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	for i := 0; i < 12; i++ {
		createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: fmt.Sprintf("task %02d", i)})
	}

	res, err := uc.List(context.Background(), model.ListTasksInput{UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, int32(1), res.Page)
	assert.Equal(t, int32(10), res.Limit)
	assert.Len(t, res.Tasks, 10)
	assert.Equal(t, int64(12), res.Total)
	assert.Equal(t, int32(2), res.TotalPages)

	second, err := uc.List(context.Background(), model.ListTasksInput{UserID: "u1", Page: 2, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, second.Tasks, 2)
	assert.Equal(t, int32(2), second.Page)

	clamped, err := uc.List(context.Background(), model.ListTasksInput{UserID: "u1", Limit: 500})
	require.NoError(t, err)
	assert.Equal(t, int32(100), clamped.Limit)
}

func TestTaskListFiltersByStatusSlug(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	todo := mustCreateStatusID(t, statuses, "u1", "todo")
	done := mustCreateStatusID(t, statuses, "u1", "done")

	createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", StatusID: todo, Title: "alpha"})
	createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", StatusID: done, Title: "beta"})

	res, err := uc.List(context.Background(), model.ListTasksInput{UserID: "u1", Status: "done"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	require.Len(t, res.Tasks, 1)
	assert.Equal(t, "beta", res.Tasks[0].Title)
}

func TestTaskListUnknownStatusIsEmpty(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "alpha"})

	res, err := uc.List(context.Background(), model.ListTasksInput{UserID: "u1", Status: "nope"})
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.Total)
	assert.Empty(t, res.Tasks)
}

func TestTaskListSearchesTitleCaseInsensitive(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "Write docs"})
	createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "Fix bug"})

	res, err := uc.List(context.Background(), model.ListTasksInput{UserID: "u1", Search: "DOCS"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	require.Len(t, res.Tasks, 1)
	assert.Equal(t, "Write docs", res.Tasks[0].Title)
}

func TestTaskListScopedToUser(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	seedDefaults(t, statuses, "u2")
	teams.addMember("t1", "u1")
	teams.addMember("t1", "u2")

	createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "mine"})
	createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u2", TeamID: "t1", Title: "theirs"})

	res, err := uc.List(context.Background(), model.ListTasksInput{UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	require.Len(t, res.Tasks, 1)
	assert.Equal(t, "mine", res.Tasks[0].Title)
}

func TestTaskUpdateChangesFields(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "old"})

	desc := "new desc"
	done := mustCreateStatusID(t, statuses, "u1", "done")

	got, err := uc.Update(context.Background(), model.UpdateTaskInput{
		UserID: "u1", TaskID: created.ID, Title: ptr("new"), Description: &desc, StatusID: &done,
	})
	require.NoError(t, err)
	assert.Equal(t, "new", got.Title)
	require.NotNil(t, got.Description)
	assert.Equal(t, "new desc", *got.Description)
	assert.Equal(t, done, got.StatusID)
}

func TestTaskUpdateRejectsEmptyTitle(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "old"})

	_, err := uc.Update(context.Background(), model.UpdateTaskInput{UserID: "u1", TaskID: created.ID, Title: ptr("   ")})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeValidation, appErr.Code)
}

func TestTaskUpdateByNonOwnerIsNotFound(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "old"})

	_, err := uc.Update(context.Background(), model.UpdateTaskInput{UserID: "u2", TaskID: created.ID, Title: ptr("hijack")})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeNotFound, appErr.Code)
}

func TestTaskUpdateRejectsForeignStatus(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	seedDefaults(t, statuses, "u2")
	teams.addMember("t1", "u1")
	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "old"})
	foreign := mustCreateStatusID(t, statuses, "u2", "done")

	_, err := uc.Update(context.Background(), model.UpdateTaskInput{UserID: "u1", TaskID: created.ID, StatusID: &foreign})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeValidation, appErr.Code)
}

func TestTaskDeleteSoftDeletes(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "gone"})

	require.NoError(t, uc.Delete(context.Background(), "u1", created.ID))

	_, err := uc.Get(context.Background(), "u1", created.ID)
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeNotFound, appErr.Code, "deleted task should not be readable")

	res, err := uc.List(context.Background(), model.ListTasksInput{UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.Total, "deleted task should not appear in listings")
}

func TestTaskDeleteByNonOwnerIsNotFound(t *testing.T) {
	uc, statuses, teams := newTaskUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "keep"})

	err := uc.Delete(context.Background(), "u2", created.ID)
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeNotFound, appErr.Code)
}

func newAssignUsecase() (*TaskUsecase, *fakeStatusRepo, *fakeTeamRepo, *fakeTaskLogRepo, *fakeNotifier) {
	tasks := newFakeTaskRepo()
	statuses := &fakeStatusRepo{inUse: map[string]bool{}}
	teams := newFakeTeamRepo()
	idempotency := newFakeIdempotencyRepo()
	logs := &fakeTaskLogRepo{}
	notifier := &fakeNotifier{}
	txm := &txManager{tasks: tasks, idempotency: idempotency, taskLogs: logs}

	uc := NewTaskUsecase(tasks, statuses, teams, idempotency, logs, notifier, txm, fakeTaskEncoder)
	return uc, statuses, teams, logs, notifier
}

func TestTaskAssignSuccess(t *testing.T) {
	uc, statuses, teams, logs, notifier := newAssignUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	teams.addMember("t1", "u2")
	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "T"})

	assigned, err := uc.Assign(context.Background(), model.AssignTaskInput{UserID: "u1", TaskID: created.ID, AssigneeID: "u2"})
	require.NoError(t, err)
	require.NotNil(t, assigned.AssigneeID)
	assert.Equal(t, "u2", *assigned.AssigneeID)
	assert.Equal(t, 1, logs.count(), "assignment must be audited")
	assert.Equal(t, 1, notifier.callCount(), "assignee must be notified")
}

func TestTaskAssignToSelfIsAllowed(t *testing.T) {
	uc, statuses, teams, _, _ := newAssignUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "T"})

	assigned, err := uc.Assign(context.Background(), model.AssignTaskInput{UserID: "u1", TaskID: created.ID, AssigneeID: "u1"})
	require.NoError(t, err)
	require.NotNil(t, assigned.AssigneeID)
	assert.Equal(t, "u1", *assigned.AssigneeID)
}

func TestTaskAssignNonMemberIsForbidden(t *testing.T) {
	uc, statuses, teams, logs, notifier := newAssignUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "T"})

	_, err := uc.Assign(context.Background(), model.AssignTaskInput{UserID: "u1", TaskID: created.ID, AssigneeID: "u3"})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeForbidden, appErr.Code)
	assert.Equal(t, 0, logs.count())
	assert.Equal(t, 0, notifier.callCount())
}

func TestTaskAssignByNonCreatorIsNotFound(t *testing.T) {
	uc, statuses, teams, _, _ := newAssignUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	teams.addMember("t1", "u2")
	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "T"})

	_, err := uc.Assign(context.Background(), model.AssignTaskInput{UserID: "u2", TaskID: created.ID, AssigneeID: "u2"})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeNotFound, appErr.Code)
}

func TestTaskAssignRollsBackWhenNotifierFails(t *testing.T) {
	uc, statuses, teams, logs, notifier := newAssignUsecase()
	seedDefaults(t, statuses, "u1")
	teams.addMember("t1", "u1")
	teams.addMember("t1", "u2")
	created := createTaskForTest(t, uc, model.CreateTaskInput{UserID: "u1", TeamID: "t1", Title: "T"})
	notifier.err = errors.New("smtp down")

	_, err := uc.Assign(context.Background(), model.AssignTaskInput{UserID: "u1", TaskID: created.ID, AssigneeID: "u2"})
	require.Error(t, err)

	got, err := uc.Get(context.Background(), "u1", created.ID)
	require.NoError(t, err)
	assert.Nil(t, got.AssigneeID, "assignee update must be rolled back")
	assert.Equal(t, 0, logs.count(), "audit log must be rolled back")
}
