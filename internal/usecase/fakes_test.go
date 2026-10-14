package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/haryfun/task-management-service/internal/domain"
)

// ---- user repository ----

type fakeUserRepo struct {
	mu     sync.Mutex
	byID   map[string]*domain.User
	byMail map[string]*domain.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byID: map[string]*domain.User{}, byMail: map[string]*domain.User{}}
}

func (r *fakeUserRepo) Create(_ context.Context, u *domain.User) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byMail[u.Email]; ok {
		return nil, domain.ErrConflict.WithMessage("email already registered")
	}
	cp := *u
	cp.ID = uuid.NewString()
	r.byID[cp.ID] = &cp
	r.byMail[cp.Email] = &cp
	return &cp, nil
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, ok := r.byMail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *fakeUserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

var _ domain.UserRepository = (*fakeUserRepo)(nil)

// ---- task status repository ----

type fakeStatusRepo struct {
	mu       sync.Mutex
	statuses []domain.TaskStatus
	inUse    map[string]bool
}

func (r *fakeStatusRepo) Create(_ context.Context, s *domain.TaskStatus) (*domain.TaskStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.statuses {
		if existing.UserID == s.UserID && existing.Slug == s.Slug {
			return nil, domain.ErrConflict
		}
	}
	cp := *s
	cp.ID = uuid.NewString()
	r.statuses = append(r.statuses, cp)
	return &cp, nil
}

func (r *fakeStatusRepo) ListByUser(_ context.Context, userID string) ([]domain.TaskStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var out []domain.TaskStatus
	for _, s := range r.statuses {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	// Stable sort keeps insertion order for equal positions, so tests are
	// deterministic.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out, nil
}

func (r *fakeStatusRepo) GetByID(_ context.Context, userID, id string) (*domain.TaskStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range r.statuses {
		if s.UserID == userID && s.ID == id {
			cp := s
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeStatusRepo) GetBySlug(_ context.Context, userID, slug string) (*domain.TaskStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range r.statuses {
		if s.UserID == userID && s.Slug == slug {
			cp := s
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeStatusRepo) GetFirst(_ context.Context, userID string) (*domain.TaskStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var best *domain.TaskStatus
	for i := range r.statuses {
		s := r.statuses[i]
		if s.UserID != userID {
			continue
		}
		if best == nil || s.Position < best.Position {
			cp := s
			best = &cp
		}
	}
	if best == nil {
		return nil, domain.ErrNotFound
	}
	return best, nil
}

func (r *fakeStatusRepo) CountByUser(_ context.Context, userID string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var count int64
	for _, s := range r.statuses {
		if s.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (r *fakeStatusRepo) MaxPosition(_ context.Context, userID string) (int32, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	max := int32(-1)
	for _, s := range r.statuses {
		if s.UserID == userID && s.Position > max {
			max = s.Position
		}
	}
	return max, nil
}

func (r *fakeStatusRepo) Update(_ context.Context, s *domain.TaskStatus) (*domain.TaskStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.statuses {
		if r.statuses[i].UserID == s.UserID && r.statuses[i].ID == s.ID {
			cp := *s
			cp.CreatedAt = r.statuses[i].CreatedAt
			r.statuses[i] = cp
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeStatusRepo) SetPosition(_ context.Context, userID, id string, position int32) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.statuses {
		if r.statuses[i].UserID == userID && r.statuses[i].ID == id {
			r.statuses[i].Position = position
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *fakeStatusRepo) Delete(_ context.Context, userID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.statuses {
		if r.statuses[i].UserID == userID && r.statuses[i].ID == id {
			r.statuses = append(r.statuses[:i], r.statuses[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *fakeStatusRepo) IsInUse(_ context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inUse[id], nil
}

var _ domain.TaskStatusRepository = (*fakeStatusRepo)(nil)

// ---- task repository ----

type fakeTaskRepo struct {
	mu    sync.Mutex
	tasks map[string]*domain.Task
}

func newFakeTaskRepo() *fakeTaskRepo {
	return &fakeTaskRepo{tasks: map[string]*domain.Task{}}
}

func (r *fakeTaskRepo) Create(_ context.Context, t *domain.Task) (*domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := *t
	cp.ID = uuid.NewString()
	r.tasks[cp.ID] = &cp
	return &cp, nil
}

func (r *fakeTaskRepo) GetByID(_ context.Context, id string) (*domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.tasks[id]
	// Mirrors the SQL: soft-deleted rows are invisible.
	if !ok || t.DeletedAt != nil {
		return nil, domain.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *fakeTaskRepo) List(_ context.Context, f domain.TaskFilter) ([]domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	matched := r.matchingLocked(f)
	sort.Slice(matched, func(i, j int) bool { return matched[i].ID < matched[j].ID })

	start := int(f.Offset)
	if start > len(matched) {
		start = len(matched)
	}
	end := start + int(f.Limit)
	if f.Limit <= 0 || end > len(matched) {
		end = len(matched)
	}
	return append([]domain.Task{}, matched[start:end]...), nil
}

func (r *fakeTaskRepo) Count(_ context.Context, f domain.TaskFilter) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return int64(len(r.matchingLocked(f))), nil
}

// matchingLocked must be called with r.mu held.
func (r *fakeTaskRepo) matchingLocked(f domain.TaskFilter) []domain.Task {
	var out []domain.Task
	for _, t := range r.tasks {
		if t.DeletedAt != nil {
			continue
		}
		if t.CreatorID != f.UserID && (t.AssigneeID == nil || *t.AssigneeID != f.UserID) {
			continue
		}
		if f.StatusID != "" && t.StatusID != f.StatusID {
			continue
		}
		if f.Search != "" && !strings.Contains(strings.ToLower(t.Title), strings.ToLower(f.Search)) {
			continue
		}
		out = append(out, *t)
	}
	return out
}

func (r *fakeTaskRepo) Update(_ context.Context, t *domain.Task) (*domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.tasks[t.ID]
	if !ok || existing.CreatorID != t.CreatorID {
		return nil, domain.ErrNotFound
	}
	cp := *t
	r.tasks[cp.ID] = &cp
	return &cp, nil
}

func (r *fakeTaskRepo) Assign(_ context.Context, taskID, creatorID, assigneeID string) (*domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.tasks[taskID]
	if !ok || existing.DeletedAt != nil || existing.CreatorID != creatorID {
		return nil, domain.ErrNotFound
	}
	assignee := assigneeID
	existing.AssigneeID = &assignee
	existing.UpdatedAt = time.Now()
	cp := *existing
	return &cp, nil
}

func (r *fakeTaskRepo) SoftDelete(_ context.Context, id, creatorID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.tasks[id]
	if !ok || existing.CreatorID != creatorID {
		return domain.ErrNotFound
	}
	now := time.Now()
	existing.DeletedAt = &now
	return nil
}

var _ domain.TaskRepository = (*fakeTaskRepo)(nil)

// ---- team repository ----

type fakeTeamRepo struct {
	members map[string]bool
}

func newFakeTeamRepo() *fakeTeamRepo {
	return &fakeTeamRepo{members: map[string]bool{}}
}

func (r *fakeTeamRepo) addMember(teamID, userID string) {
	r.members[teamID+":"+userID] = true
}

func (r *fakeTeamRepo) IsMember(_ context.Context, teamID, userID string) (bool, error) {
	return r.members[teamID+":"+userID], nil
}

var _ domain.TeamRepository = (*fakeTeamRepo)(nil)

// ---- idempotency repository ----

type fakeIdempotencyRepo struct {
	mu      sync.Mutex
	records map[string]*domain.IdempotencyRecord
}

func newFakeIdempotencyRepo() *fakeIdempotencyRepo {
	return &fakeIdempotencyRepo{records: map[string]*domain.IdempotencyRecord{}}
}

func (r *fakeIdempotencyRepo) Get(_ context.Context, key string) (*domain.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.records[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *record
	return &cp, nil
}

func (r *fakeIdempotencyRepo) Create(_ context.Context, record *domain.IdempotencyRecord) (*domain.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.records[record.Key]; ok {
		return nil, domain.ErrConflict
	}
	cp := *record
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	r.records[cp.Key] = &cp
	return &cp, nil
}

func (r *fakeIdempotencyRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.records, key)
	return nil
}

var _ domain.IdempotencyRepository = (*fakeIdempotencyRepo)(nil)

// fakeTaskEncoder mimics the transport encoder: small JSON body + 201.
func fakeTaskEncoder(task *domain.Task) (int, []byte, error) {
	body, err := json.Marshal(map[string]string{"id": task.ID, "title": task.Title})
	return 201, body, err
}

// snapshot/restore give the fakes database-like transaction semantics.
func (r *fakeTaskRepo) snapshot() map[string]domain.Task {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make(map[string]domain.Task, len(r.tasks))
	for id, task := range r.tasks {
		out[id] = *task
	}
	return out
}

func (r *fakeTaskRepo) restore(snap map[string]domain.Task) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tasks = make(map[string]*domain.Task, len(snap))
	for id, task := range snap {
		cp := task
		r.tasks[id] = &cp
	}
}

func (r *fakeIdempotencyRepo) snapshot() map[string]domain.IdempotencyRecord {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make(map[string]domain.IdempotencyRecord, len(r.records))
	for key, record := range r.records {
		out[key] = *record
	}
	return out
}

func (r *fakeIdempotencyRepo) restore(snap map[string]domain.IdempotencyRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.records = make(map[string]*domain.IdempotencyRecord, len(snap))
	for key, record := range snap {
		cp := record
		r.records[key] = &cp
	}
}

// txManager is a transactional TxManager over the in-memory fakes: it serializes
// transactions and rolls back by restoring the pre-transaction snapshot. It lets
// cold concurrency tests exercise the same "commit both or neither" guarantee as
// Postgres, without a database.
type txManager struct {
	mu          sync.Mutex
	tasks       *fakeTaskRepo
	idempotency *fakeIdempotencyRepo
	taskLogs    *fakeTaskLogRepo
}

func (m *txManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasksSnap := m.tasks.snapshot()
	idempotencySnap := m.idempotency.snapshot()
	var logsSnap []domain.TaskLog
	if m.taskLogs != nil {
		logsSnap = m.taskLogs.snapshot()
	}

	if err := fn(ctx); err != nil {
		m.tasks.restore(tasksSnap)
		m.idempotency.restore(idempotencySnap)
		if m.taskLogs != nil {
			m.taskLogs.restore(logsSnap)
		}
		return err
	}
	return nil
}

var _ domain.TxManager = (*txManager)(nil)

// ---- task log repository ----

type fakeTaskLogRepo struct {
	mu   sync.Mutex
	logs []domain.TaskLog
}

func (r *fakeTaskLogRepo) Create(_ context.Context, log *domain.TaskLog) (*domain.TaskLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := *log
	cp.ID = uuid.NewString()
	cp.CreatedAt = time.Now()
	r.logs = append(r.logs, cp)
	return &cp, nil
}

func (r *fakeTaskLogRepo) snapshot() []domain.TaskLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.TaskLog{}, r.logs...)
}

func (r *fakeTaskLogRepo) restore(snap []domain.TaskLog) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append([]domain.TaskLog{}, snap...)
}

func (r *fakeTaskLogRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.logs)
}

var _ domain.TaskLogRepository = (*fakeTaskLogRepo)(nil)

// ---- notifier ----

type fakeNotifier struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (n *fakeNotifier) NotifyAssigned(context.Context, AssignmentNotification) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.calls++
	return n.err
}

func (n *fakeNotifier) callCount() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.calls
}

var _ Notifier = (*fakeNotifier)(nil)

// ---- cross-cutting fakes ----

type fakeTxManager struct{}

func (fakeTxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

var _ domain.TxManager = fakeTxManager{}

type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) { return "hashed:" + password, nil }

func (fakeHasher) Compare(hash, password string) error {
	if hash != "hashed:"+password {
		return errors.New("password mismatch")
	}
	return nil
}

var _ PasswordHasher = fakeHasher{}

type fakeTokenIssuer struct{}

func (fakeTokenIssuer) Issue(userID string) (string, int64, error) {
	return "token-for-" + userID, 3600, nil
}

var _ TokenIssuer = fakeTokenIssuer{}
