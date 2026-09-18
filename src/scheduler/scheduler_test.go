package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Timezone != "America/New_York" {
		t.Errorf("Timezone = %q", cfg.Timezone)
	}
	if cfg.CatchUpWindow != time.Hour {
		t.Errorf("CatchUpWindow = %v", cfg.CatchUpWindow)
	}
}

func TestNew(t *testing.T) {
	s := New("UTC")
	if s.timezone != time.UTC {
		t.Errorf("timezone = %v, want UTC", s.timezone)
	}
}

func TestNewWithConfig_InvalidTimezoneFallsBackToUTC(t *testing.T) {
	s := NewWithConfig(Config{Timezone: "Not/A/Real/Zone"})
	if s.timezone != time.UTC {
		t.Errorf("expected fallback to UTC for invalid timezone, got %v", s.timezone)
	}
}

func TestAddTask(t *testing.T) {
	s := New("UTC")
	s.AddTask("task1", "@hourly", true, func(context.Context) error { return nil })
	if s.TaskCount() != 1 {
		t.Fatalf("TaskCount = %d, want 1", s.TaskCount())
	}
	task, ok := s.GetTask("task1")
	if !ok {
		t.Fatal("expected task1 to exist")
	}
	if task.LastStatus != StatusPending {
		t.Errorf("LastStatus = %q, want pending", task.LastStatus)
	}
	if task.NextRun.IsZero() {
		t.Error("NextRun should be calculated")
	}
}

func TestAddTaskWithOptions_DefaultsIDToName(t *testing.T) {
	s := New("UTC")
	s.AddTaskWithOptions(&Task{Name: "myname", Schedule: "@daily", Enabled: true})
	if _, ok := s.GetTask("myname"); !ok {
		t.Error("expected task registered under Name when ID is empty")
	}
}

func TestGetTask_NotFound(t *testing.T) {
	s := New("UTC")
	if _, ok := s.GetTask("missing"); ok {
		t.Error("expected ok=false for missing task")
	}
}

func TestGetTask_ReturnsCopy(t *testing.T) {
	s := New("UTC")
	s.AddTask("t", "@daily", true, func(context.Context) error { return nil })
	task, _ := s.GetTask("t")
	task.Name = "mutated"

	original, _ := s.GetTask("t")
	if original.Name == "mutated" {
		t.Error("GetTask should return an independent copy")
	}
}

func TestEnableDisableTask(t *testing.T) {
	s := New("UTC")
	s.AddTask("t", "@daily", false, func(context.Context) error { return nil })

	s.EnableTask("t")
	task, _ := s.GetTask("t")
	if !task.Enabled {
		t.Error("expected task enabled")
	}

	s.DisableTask("t")
	task, _ = s.GetTask("t")
	if task.Enabled {
		t.Error("expected task disabled")
	}

	// no-op on missing task should not panic
	s.EnableTask("missing")
	s.DisableTask("missing")
}

func TestUpdateSchedule(t *testing.T) {
	s := New("UTC")
	s.AddTask("t", "@daily", true, func(context.Context) error { return nil })

	if err := s.UpdateSchedule("t", "@hourly"); err != nil {
		t.Fatalf("UpdateSchedule: %v", err)
	}
	task, _ := s.GetTask("t")
	if task.Schedule != "@hourly" {
		t.Errorf("Schedule = %q, want @hourly", task.Schedule)
	}
}

func TestUpdateSchedule_NotFound(t *testing.T) {
	s := New("UTC")
	if err := s.UpdateSchedule("missing", "@hourly"); err == nil {
		t.Error("expected error for missing task")
	}
}

func TestUpdateSchedule_InvalidSchedule(t *testing.T) {
	s := New("UTC")
	s.AddTask("t", "@daily", true, func(context.Context) error { return nil })
	if err := s.UpdateSchedule("t", "not a valid cron"); err == nil {
		t.Error("expected error for invalid schedule")
	}
}

func TestRunNow(t *testing.T) {
	s := New("UTC")
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()
	done := make(chan struct{})
	s.AddTask("t", "@daily", true, func(context.Context) error {
		close(done)
		return nil
	})

	if err := s.RunNow("t"); err != nil {
		t.Fatalf("RunNow: %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("task did not run in time")
	}
}

func TestRunNow_NotFound(t *testing.T) {
	s := New("UTC")
	if err := s.RunNow("missing"); err == nil {
		t.Error("expected error for missing task")
	}
}

func TestRunTask_SuccessUpdatesState(t *testing.T) {
	s := New("UTC")
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	task := &Task{ID: "t", Name: "t", Schedule: "@hourly", Func: func(context.Context) error { return nil }}
	s.runTask(task)

	if task.LastStatus != StatusSuccess {
		t.Errorf("LastStatus = %q, want success", task.LastStatus)
	}
	if task.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", task.RunCount)
	}
	if task.Running {
		t.Error("Running should be false after completion")
	}
}

func TestRunTask_FailureUpdatesState(t *testing.T) {
	s := New("UTC")
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	wantErr := errors.New("boom")
	task := &Task{ID: "t", Name: "t", Schedule: "@hourly", Func: func(context.Context) error { return wantErr }}
	s.runTask(task)

	if task.LastStatus != StatusFailed {
		t.Errorf("LastStatus = %q, want failed", task.LastStatus)
	}
	if task.LastError != "boom" {
		t.Errorf("LastError = %q, want boom", task.LastError)
	}
	if task.FailCount != 1 {
		t.Errorf("FailCount = %d, want 1", task.FailCount)
	}
}

func TestRunTask_RetryOnFail(t *testing.T) {
	s := New("UTC")
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	task := &Task{
		ID:          "t",
		Name:        "t",
		Schedule:    "@hourly",
		RetryOnFail: true,
		RetryDelay:  5 * time.Minute,
		Func:        func(context.Context) error { return errors.New("fail") },
	}
	before := time.Now()
	s.runTask(task)

	if task.NextRun.Sub(before) > 6*time.Minute || task.NextRun.Sub(before) < 4*time.Minute {
		t.Errorf("NextRun not scheduled via retry delay: %v", task.NextRun)
	}
}

func TestStartStop(t *testing.T) {
	s := New("UTC")
	s.Start()
	if !s.IsRunning() {
		t.Error("expected scheduler running after Start")
	}
	// starting again should be a no-op, not block or panic
	s.Start()
	s.Stop()
	if s.IsRunning() {
		t.Error("expected scheduler stopped after Stop")
	}
	// stopping again should be a no-op
	s.Stop()
}

func TestCalculateNextRun_Every(t *testing.T) {
	s := New("UTC")
	from := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	next, err := s.calculateNextRun("@every 5m", from)
	if err != nil {
		t.Fatalf("calculateNextRun: %v", err)
	}
	want := from.Add(5 * time.Minute)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestCalculateNextRun_EveryInvalidDuration(t *testing.T) {
	s := New("UTC")
	if _, err := s.calculateNextRun("@every notaduration", time.Now()); err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestCalculateNextRun_Hourly(t *testing.T) {
	s := New("UTC")
	from := time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC)
	next, err := s.calculateNextRun("@hourly", from)
	if err != nil {
		t.Fatalf("calculateNextRun: %v", err)
	}
	want := time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestCalculateNextRun_Daily(t *testing.T) {
	s := New("UTC")
	from := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	next, err := s.calculateNextRun("@daily", from)
	if err != nil {
		t.Fatalf("calculateNextRun: %v", err)
	}
	want := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestCalculateNextRun_Weekly(t *testing.T) {
	s := New("UTC")
	// 2024-01-01 is a Monday.
	from := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	next, err := s.calculateNextRun("@weekly", from)
	if err != nil {
		t.Fatalf("calculateNextRun: %v", err)
	}
	if next.Weekday() != time.Sunday {
		t.Errorf("expected next Sunday, got %v", next.Weekday())
	}
	if !next.After(from) {
		t.Error("expected next weekly run to be after from")
	}
}

func TestCalculateNextRun_Monthly(t *testing.T) {
	s := New("UTC")
	from := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	next, err := s.calculateNextRun("@monthly", from)
	if err != nil {
		t.Fatalf("calculateNextRun: %v", err)
	}
	want := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestCalculateNextRun_CronExpression(t *testing.T) {
	s := New("UTC")
	from := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	// Every day at 15:30
	next, err := s.calculateNextRun("30 15 * * *", from)
	if err != nil {
		t.Fatalf("calculateNextRun: %v", err)
	}
	want := time.Date(2024, 1, 1, 15, 30, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestParseDuration(t *testing.T) {
	cases := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"5s", 5 * time.Second, false},
		{"5m", 5 * time.Minute, false},
		{"5h", 5 * time.Hour, false},
		{"2d", 48 * time.Hour, false},
		{"5x", 0, true},
		{"x", 0, true},
		{"", 0, true},
	}
	for _, c := range cases {
		got, err := parseDuration(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseDuration(%q) expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDuration(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseDuration(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseCronExpression_WrongFieldCount(t *testing.T) {
	if _, err := parseCronExpression("* * *", time.Now(), time.UTC); err == nil {
		t.Error("expected error for wrong field count")
	}
}

func TestParseCronExpression_InvalidField(t *testing.T) {
	cases := []string{
		"xx * * * *",
		"* xx * * *",
		"* * xx * *",
		"* * * xx *",
		"* * * * xx",
	}
	for _, expr := range cases {
		if _, err := parseCronExpression(expr, time.Now(), time.UTC); err == nil {
			t.Errorf("expected error for invalid cron %q", expr)
		}
	}
}

func TestParseCronField(t *testing.T) {
	all, err := parseCronField("*", 0, 3)
	if err != nil || len(all) != 4 {
		t.Errorf("parseCronField(*) = %v, %v", all, err)
	}

	step, err := parseCronField("*/2", 0, 6)
	if err != nil {
		t.Fatalf("parseCronField(*/2): %v", err)
	}
	wantStep := []int{0, 2, 4, 6}
	if len(step) != len(wantStep) {
		t.Fatalf("parseCronField(*/2) = %v, want %v", step, wantStep)
	}
	for i := range wantStep {
		if step[i] != wantStep[i] {
			t.Errorf("parseCronField(*/2)[%d] = %d, want %d", i, step[i], wantStep[i])
		}
	}

	rng, err := parseCronField("1-3", 0, 10)
	if err != nil || len(rng) != 3 {
		t.Errorf("parseCronField(1-3) = %v, %v", rng, err)
	}

	list, err := parseCronField("1,3,5", 0, 10)
	if err != nil || len(list) != 3 {
		t.Errorf("parseCronField(1,3,5) = %v, %v", list, err)
	}

	single, err := parseCronField("7", 0, 10)
	if err != nil || len(single) != 1 || single[0] != 7 {
		t.Errorf("parseCronField(7) = %v, %v", single, err)
	}

	if _, err := parseCronField("*/bad", 0, 10); err == nil {
		t.Error("expected error for */bad")
	}
	if _, err := parseCronField("1-2-3", 0, 10); err == nil {
		t.Error("expected error for malformed range")
	}
	if _, err := parseCronField("a-3", 0, 10); err == nil {
		t.Error("expected error for non-numeric range start")
	}
	if _, err := parseCronField("1-b", 0, 10); err == nil {
		t.Error("expected error for non-numeric range end")
	}
	if _, err := parseCronField("notanumber", 0, 10); err == nil {
		t.Error("expected error for non-numeric field")
	}
}

func TestMatches(t *testing.T) {
	if !matches(5, []int{1, 5, 9}) {
		t.Error("expected match")
	}
	if matches(6, []int{1, 5, 9}) {
		t.Error("expected no match")
	}
}

func TestGetTasks(t *testing.T) {
	s := New("UTC")
	s.AddTask("a", "@daily", true, func(context.Context) error { return nil })
	s.AddTask("b", "@daily", true, func(context.Context) error { return nil })

	tasks := s.GetTasks()
	if len(tasks) != 2 {
		t.Errorf("GetTasks length = %d, want 2", len(tasks))
	}
}

func TestCheckCatchUp_RunsMissedTasks(t *testing.T) {
	s := New("UTC")
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	ran := make(chan struct{}, 1)
	s.AddTask("t", "@hourly", true, func(context.Context) error {
		ran <- struct{}{}
		return nil
	})
	// Force the task to look overdue but within the catch-up window.
	task, _ := s.GetTask("t")
	_ = task
	s.mu.Lock()
	s.tasks["t"].NextRun = time.Now().In(s.timezone).Add(-time.Minute)
	s.mu.Unlock()

	s.checkCatchUp()

	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("expected catch-up task to run")
	}
}
