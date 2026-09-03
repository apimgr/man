// Package scheduler provides a built-in task scheduler for casman.
// See AI.md PART 19 for details.
// The scheduler is ALWAYS running - no external cron needed.
package scheduler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TaskStatus represents the result of a task execution.
type TaskStatus string

const (
	StatusPending TaskStatus = "pending"
	StatusRunning TaskStatus = "running"
	StatusSuccess TaskStatus = "success"
	StatusFailed  TaskStatus = "failed"
	StatusSkipped TaskStatus = "skipped"
)

// Task represents a scheduled task.
type Task struct {
	ID          string
	Name        string
	Schedule    string
	Enabled     bool
	Func        func(context.Context) error
	LastRun     time.Time
	LastStatus  TaskStatus
	LastError   string
	NextRun     time.Time
	RunCount    int64
	FailCount   int64
	Running     bool
	RetryOnFail bool
	RetryDelay  time.Duration
}

// Config holds scheduler configuration.
type Config struct {
	Timezone      string
	CatchUpWindow time.Duration
}

// DefaultConfig returns default scheduler configuration.
func DefaultConfig() Config {
	return Config{
		Timezone:      "America/New_York",
		CatchUpWindow: time.Hour,
	}
}

// Scheduler manages scheduled tasks.
type Scheduler struct {
	config   Config
	tasks    map[string]*Task
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	timezone *time.Location
	running  bool
}

// New creates a new Scheduler.
func New(timezone string) *Scheduler {
	cfg := DefaultConfig()
	cfg.Timezone = timezone
	return NewWithConfig(cfg)
}

// NewWithConfig creates a new Scheduler with configuration.
func NewWithConfig(cfg Config) *Scheduler {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		loc = time.UTC
	}

	return &Scheduler{
		config:   cfg,
		tasks:    make(map[string]*Task),
		timezone: loc,
	}
}

// AddTask adds a task to the scheduler.
func (s *Scheduler) AddTask(name, schedule string, enabled bool, fn func(context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := &Task{
		ID:         name,
		Name:       name,
		Schedule:   schedule,
		Enabled:    enabled,
		Func:       fn,
		LastStatus: StatusPending,
	}

	// Calculate initial next run
	now := time.Now().In(s.timezone)
	nextRun, err := s.calculateNextRun(schedule, now)
	if err == nil {
		task.NextRun = nextRun
	}

	s.tasks[name] = task
}

// AddTaskWithOptions adds a task with additional options.
func (s *Scheduler) AddTaskWithOptions(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task.ID == "" {
		task.ID = task.Name
	}
	task.LastStatus = StatusPending

	// Calculate initial next run
	now := time.Now().In(s.timezone)
	nextRun, err := s.calculateNextRun(task.Schedule, now)
	if err == nil {
		task.NextRun = nextRun
	}

	s.tasks[task.ID] = task
}

// Start starts the scheduler.
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.running = true
	s.mu.Unlock()

	// Check for catch-up tasks
	s.checkCatchUp()

	s.wg.Add(1)
	go s.run()

	log.Printf("Scheduler started with %d tasks", len(s.tasks))
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.cancel()
	s.mu.Unlock()

	s.wg.Wait()
	log.Println("Scheduler stopped")
}

// checkCatchUp runs missed tasks within the catch-up window.
func (s *Scheduler) checkCatchUp() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().In(s.timezone)
	cutoff := now.Add(-s.config.CatchUpWindow)

	for _, task := range s.tasks {
		if !task.Enabled {
			continue
		}

		// If next run is in the past but within catch-up window
		if task.NextRun.Before(now) && task.NextRun.After(cutoff) {
			log.Printf("Scheduler: catch-up queuing missed task %s", task.Name)
			go s.runTask(task)
		}
	}
}

// run is the main scheduler loop.
func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkTasks()
		}
	}
}

// checkTasks checks if any tasks need to run.
func (s *Scheduler) checkTasks() {
	s.mu.RLock()
	now := time.Now().In(s.timezone)
	var toRun []*Task

	for _, task := range s.tasks {
		if !task.Enabled || task.Running {
			continue
		}

		if task.NextRun.Before(now) || task.NextRun.Equal(now) {
			toRun = append(toRun, task)
		}
	}
	s.mu.RUnlock()

	for _, task := range toRun {
		go s.runTask(task)
	}
}

// runTask executes a task.
func (s *Scheduler) runTask(task *Task) {
	s.mu.Lock()
	task.Running = true
	task.LastStatus = StatusRunning
	s.mu.Unlock()

	log.Printf("Scheduler: running task %s", task.Name)

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
	defer cancel()

	err := task.Func(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().In(s.timezone)
	task.Running = false
	task.LastRun = now

	if err != nil {
		task.LastStatus = StatusFailed
		task.LastError = err.Error()
		task.FailCount++
		log.Printf("Scheduler: task %s failed: %v", task.Name, err)

		// Handle retry
		if task.RetryOnFail && task.RetryDelay > 0 {
			task.NextRun = now.Add(task.RetryDelay)
		} else {
			nextRun, _ := s.calculateNextRun(task.Schedule, now)
			task.NextRun = nextRun
		}
	} else {
		task.LastStatus = StatusSuccess
		task.LastError = ""
		task.RunCount++
		log.Printf("Scheduler: task %s completed", task.Name)

		nextRun, _ := s.calculateNextRun(task.Schedule, now)
		task.NextRun = nextRun
	}
}

// calculateNextRun calculates when a task should next run.
func (s *Scheduler) calculateNextRun(schedule string, from time.Time) (time.Time, error) {
	// Handle @every interval
	if strings.HasPrefix(schedule, "@every ") {
		durStr := strings.TrimPrefix(schedule, "@every ")
		dur, err := parseDuration(durStr)
		if err != nil {
			return time.Time{}, err
		}
		return from.Add(dur), nil
	}

	// Handle special schedules
	switch schedule {
	case "@hourly":
		return from.Truncate(time.Hour).Add(time.Hour), nil
	case "@daily":
		next := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, s.timezone)
		if !next.After(from) {
			next = next.AddDate(0, 0, 1)
		}
		return next, nil
	case "@weekly":
		next := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, s.timezone)
		// Find next Sunday
		daysUntilSunday := (7 - int(next.Weekday())) % 7
		if daysUntilSunday == 0 && !next.After(from) {
			daysUntilSunday = 7
		}
		return next.AddDate(0, 0, daysUntilSunday), nil
	case "@monthly":
		next := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, s.timezone)
		if !next.After(from) {
			next = next.AddDate(0, 1, 0)
		}
		return next, nil
	}

	// Parse cron expression (minute hour day month weekday)
	return parseCronExpression(schedule, from, s.timezone)
}

// parseDuration parses a duration string like "5m", "2h".
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration number: %s", numStr)
	}

	switch unit {
	case 's':
		return time.Duration(num) * time.Second, nil
	case 'm':
		return time.Duration(num) * time.Minute, nil
	case 'h':
		return time.Duration(num) * time.Hour, nil
	case 'd':
		return time.Duration(num) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid duration unit: %c", unit)
	}
}

// parseCronExpression parses a standard cron expression.
// Format: minute hour day month weekday
func parseCronExpression(expr string, from time.Time, loc *time.Location) (time.Time, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return time.Time{}, fmt.Errorf("cron expression must have 5 fields: %s", expr)
	}

	minute, err := parseCronField(parts[0], 0, 59)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minute: %w", err)
	}
	hour, err := parseCronField(parts[1], 0, 23)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hour: %w", err)
	}
	day, err := parseCronField(parts[2], 1, 31)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %w", err)
	}
	month, err := parseCronField(parts[3], 1, 12)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month: %w", err)
	}
	weekday, err := parseCronField(parts[4], 0, 6)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid weekday: %w", err)
	}

	// Find next matching time starting from next minute
	next := from.Add(time.Minute).Truncate(time.Minute)

	// Search for up to 4 years
	maxIterations := 365 * 4 * 24 * 60
	for i := 0; i < maxIterations; i++ {
		if matches(int(next.Month()), month) &&
			matches(next.Day(), day) &&
			matches(int(next.Weekday()), weekday) &&
			matches(next.Hour(), hour) &&
			matches(next.Minute(), minute) {
			return next, nil
		}
		next = next.Add(time.Minute)
	}

	return time.Time{}, fmt.Errorf("no matching time found within 4 years")
}

// parseCronField parses a single cron field.
func parseCronField(field string, min, max int) ([]int, error) {
	if field == "*" {
		var result []int
		for i := min; i <= max; i++ {
			result = append(result, i)
		}
		return result, nil
	}

	// Handle */n syntax
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(strings.TrimPrefix(field, "*/"))
		if err != nil {
			return nil, err
		}
		var result []int
		for i := min; i <= max; i += step {
			result = append(result, i)
		}
		return result, nil
	}

	// Handle comma-separated values
	var result []int
	for _, part := range strings.Split(field, ",") {
		// Handle range
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			start, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, err
			}
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
		} else {
			val, err := strconv.Atoi(part)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		}
	}
	return result, nil
}

// matches checks if a value is in the allowed list.
func matches(val int, allowed []int) bool {
	for _, a := range allowed {
		if a == val {
			return true
		}
	}
	return false
}

// GetTasks returns all tasks.
func (s *Scheduler) GetTasks() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, *t)
	}
	return tasks
}

// GetTask returns a specific task by ID.
func (s *Scheduler) GetTask(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	if !ok {
		return nil, false
	}
	// Return a copy
	copy := *task
	return &copy, true
}

// EnableTask enables a task.
func (s *Scheduler) EnableTask(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, ok := s.tasks[name]; ok {
		task.Enabled = true
	}
}

// DisableTask disables a task.
func (s *Scheduler) DisableTask(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, ok := s.tasks[name]; ok {
		task.Enabled = false
	}
}

// RunNow runs a task immediately.
func (s *Scheduler) RunNow(name string) error {
	s.mu.RLock()
	task, ok := s.tasks[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found: %s", name)
	}

	go s.runTask(task)
	return nil
}

// UpdateSchedule updates a task's schedule.
func (s *Scheduler) UpdateSchedule(name, schedule string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[name]
	if !ok {
		return fmt.Errorf("task not found: %s", name)
	}

	// Validate schedule
	now := time.Now().In(s.timezone)
	nextRun, err := s.calculateNextRun(schedule, now)
	if err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}

	task.Schedule = schedule
	task.NextRun = nextRun
	return nil
}

// IsRunning returns whether the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// TaskCount returns the number of registered tasks.
func (s *Scheduler) TaskCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tasks)
}
