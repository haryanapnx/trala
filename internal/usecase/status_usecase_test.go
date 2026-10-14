package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/haryfun/task-management-service/internal/domain"
	"github.com/haryfun/task-management-service/internal/model"
)

func ptr[T any](v T) *T { return &v }

func newStatusUsecase() (*StatusUsecase, *fakeStatusRepo) {
	repo := &fakeStatusRepo{inUse: map[string]bool{}}
	return NewStatusUsecase(repo, fakeTxManager{}), repo
}

func mustCreateStatus(t *testing.T, repo *fakeStatusRepo, userID, slug string, position int32) *domain.TaskStatus {
	t.Helper()
	s, err := repo.Create(context.Background(), &domain.TaskStatus{
		UserID: userID, Name: slug, Slug: slug, Position: position, IsDefault: true,
	})
	require.NoError(t, err)
	return s
}

func seedDefaults(t *testing.T, repo *fakeStatusRepo, userID string) {
	t.Helper()
	mustCreateStatus(t, repo, userID, "todo", 0)
	mustCreateStatus(t, repo, userID, "in_progress", 1)
	mustCreateStatus(t, repo, userID, "done", 2)
}

func slugsOf(list []domain.TaskStatus) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		out = append(out, s.Slug)
	}
	return out
}

func positionsOf(list []domain.TaskStatus) []int32 {
	out := make([]int32, 0, len(list))
	for _, s := range list {
		out = append(out, s.Position)
	}
	return out
}

func TestStatusListScopedToUserAndOrdered(t *testing.T) {
	uc, repo := newStatusUsecase()
	seedDefaults(t, repo, "u1")
	mustCreateStatus(t, repo, "u2", "todo", 0)

	list, err := uc.List(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []string{"todo", "in_progress", "done"}, slugsOf(list))
}

func TestStatusCreateAppendsAndSlugifies(t *testing.T) {
	uc, repo := newStatusUsecase()
	seedDefaults(t, repo, "u1")

	status, err := uc.Create(context.Background(), model.CreateStatusInput{UserID: "u1", Name: "In Review"})
	require.NoError(t, err)
	assert.Equal(t, "in_review", status.Slug)
	assert.Equal(t, int32(3), status.Position)

	list, err := uc.List(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []int32{0, 1, 2, 3}, positionsOf(list))
}

func TestStatusCreateWithPositionNormalizes(t *testing.T) {
	uc, repo := newStatusUsecase()
	seedDefaults(t, repo, "u1")

	status, err := uc.Create(context.Background(), model.CreateStatusInput{
		UserID: "u1", Name: "Blocked", Position: ptr(int32(1)),
	})
	require.NoError(t, err)

	list, err := uc.List(context.Background(), "u1")
	require.NoError(t, err)
	assert.Len(t, list, 4)
	assert.Equal(t, []int32{0, 1, 2, 3}, positionsOf(list))
	assert.Equal(t, []string{"todo", "blocked", "in_progress", "done"}, slugsOf(list))
	assert.Equal(t, int32(1), status.Position)
}

func TestStatusCreateDuplicateSlugConflict(t *testing.T) {
	uc, repo := newStatusUsecase()
	seedDefaults(t, repo, "u1")

	_, err := uc.Create(context.Background(), model.CreateStatusInput{UserID: "u1", Name: "Todo"})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeConflict, appErr.Code)
	assert.Contains(t, appErr.Message, "slug")
}

func TestStatusCreateRequiresName(t *testing.T) {
	uc, _ := newStatusUsecase()

	_, err := uc.Create(context.Background(), model.CreateStatusInput{UserID: "u1", Name: "   "})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeValidation, appErr.Code)
}

func TestStatusUpdateReordersAndNormalizes(t *testing.T) {
	uc, repo := newStatusUsecase()
	seedDefaults(t, repo, "u1")
	todo := mustCreateStatusID(t, repo, "u1", "todo")

	_, err := uc.Update(context.Background(), model.UpdateStatusInput{
		UserID: "u1", StatusID: todo, Position: ptr(int32(2)),
	})
	require.NoError(t, err)

	list, err := uc.List(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []int32{0, 1, 2}, positionsOf(list))
	assert.Equal(t, []string{"in_progress", "done", "todo"}, slugsOf(list),
		"moving todo to index 2 should place it last")
}

func TestStatusUpdateUnknownIsNotFound(t *testing.T) {
	uc, _ := newStatusUsecase()

	_, err := uc.Update(context.Background(), model.UpdateStatusInput{
		UserID: "u1", StatusID: "does-not-exist", Name: ptr("x"),
	})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeNotFound, appErr.Code)
}

func TestStatusDeleteInUseConflict(t *testing.T) {
	uc, repo := newStatusUsecase()
	seedDefaults(t, repo, "u1")
	done := mustCreateStatusID(t, repo, "u1", "done")
	repo.inUse[done] = true

	err := uc.Delete(context.Background(), "u1", done)
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeConflict, appErr.Code)
	assert.Contains(t, appErr.Message, "used by")
}

func TestStatusDeleteLastStatusConflict(t *testing.T) {
	uc, repo := newStatusUsecase()
	only := mustCreateStatus(t, repo, "u1", "todo", 0)

	err := uc.Delete(context.Background(), "u1", only.ID)
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeConflict, appErr.Code)
	assert.Contains(t, appErr.Message, "last status")
}

func TestStatusDeleteNormalizesPositions(t *testing.T) {
	uc, repo := newStatusUsecase()
	seedDefaults(t, repo, "u1")
	progress := mustCreateStatusID(t, repo, "u1", "in_progress")

	require.NoError(t, uc.Delete(context.Background(), "u1", progress))

	list, err := uc.List(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []string{"todo", "done"}, slugsOf(list))
	assert.Equal(t, []int32{0, 1}, positionsOf(list))
}

func mustCreateStatusID(t *testing.T, repo *fakeStatusRepo, userID, slug string) string {
	t.Helper()
	for _, s := range repo.statuses {
		if s.UserID == userID && s.Slug == slug {
			return s.ID
		}
	}
	t.Fatalf("status %q for user %q not found", slug, userID)
	return ""
}
