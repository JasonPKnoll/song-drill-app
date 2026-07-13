package srs

import (
	"testing"
	"time"
)

var epoch = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestNew(t *testing.T) {
	s := New(epoch)
	if s.Stage != StageNew {
		t.Errorf("Stage = %q, want %q", s.Stage, StageNew)
	}
	if s.EaseFactor != StartingEase {
		t.Errorf("EaseFactor = %v, want %v", s.EaseFactor, StartingEase)
	}
	if !s.Due.Equal(epoch) {
		t.Errorf("Due = %v, want %v", s.Due, epoch)
	}
}

// A brand-new card must take exactly len(LearningSteps) consecutive correct
// answers to graduate — this is the exact behavior the "one right answer
// instantly marks it done" bug report hinges on. One correct answer must
// leave it mid-rotation, not done.
func TestAnswer_NewCardGraduatesOnlyAfterAllLearningSteps(t *testing.T) {
	s := New(epoch)
	now := epoch

	for i, step := range LearningSteps {
		s = Answer(s, true, now)
		if i < len(LearningSteps)-1 {
			if s.Stage != StageLearning {
				t.Fatalf("after correct answer %d/%d: Stage = %q, want %q (should not graduate yet)",
					i+1, len(LearningSteps), s.Stage, StageLearning)
			}
			wantDue := now.Add(time.Duration(step) * time.Minute)
			if !s.Due.Equal(wantDue) {
				t.Errorf("after correct answer %d: Due = %v, want %v", i+1, s.Due, wantDue)
			}
		} else {
			if s.Stage != StageReview {
				t.Fatalf("after final correct answer: Stage = %q, want %q", s.Stage, StageReview)
			}
			if s.IntervalDays != GraduatingIntervalDays {
				t.Errorf("IntervalDays = %v, want %v", s.IntervalDays, GraduatingIntervalDays)
			}
		}
	}
}

// A miss during learning resets to the first step rather than failing the
// card out of the learning phase entirely.
func TestAnswer_LearningMissResetsToFirstStep(t *testing.T) {
	s := New(epoch)
	s = Answer(s, true, epoch) // step 1/2, StepIndex=1

	missAt := epoch.Add(5 * time.Minute)
	s = Answer(s, false, missAt)

	if s.Stage != StageLearning {
		t.Errorf("Stage = %q, want %q", s.Stage, StageLearning)
	}
	if s.StepIndex != 0 {
		t.Errorf("StepIndex = %d, want 0", s.StepIndex)
	}
	wantDue := missAt.Add(time.Duration(LearningSteps[0]) * time.Minute)
	if !s.Due.Equal(wantDue) {
		t.Errorf("Due = %v, want %v", s.Due, wantDue)
	}
}

func TestAnswer_ReviewCorrectGrowsIntervalByEase(t *testing.T) {
	s := State{Stage: StageReview, EaseFactor: 2.5, IntervalDays: 4, Due: epoch}
	s = Answer(s, true, epoch)

	wantInterval := 4 * 2.5
	if s.IntervalDays != wantInterval {
		t.Errorf("IntervalDays = %v, want %v", s.IntervalDays, wantInterval)
	}
	if s.Stage != StageReview {
		t.Errorf("Stage = %q, want %q", s.Stage, StageReview)
	}
	wantDue := epoch.Add(daysToDuration(wantInterval))
	if !s.Due.Equal(wantDue) {
		t.Errorf("Due = %v, want %v", s.Due, wantDue)
	}
}

// A lapse in review drops ease, forfeits the earned interval down to the
// minimum, and sends the card into relearning — not back to "new".
func TestAnswer_ReviewLapseEntersRelearning(t *testing.T) {
	s := State{Stage: StageReview, EaseFactor: 2.5, IntervalDays: 30, Lapses: 0, Due: epoch}
	s = Answer(s, false, epoch)

	if s.Stage != StageRelearning {
		t.Errorf("Stage = %q, want %q", s.Stage, StageRelearning)
	}
	if s.Lapses != 1 {
		t.Errorf("Lapses = %d, want 1", s.Lapses)
	}
	wantEase := 2.5 - EaseAgainPenalty
	if s.EaseFactor != wantEase {
		t.Errorf("EaseFactor = %v, want %v", s.EaseFactor, wantEase)
	}
	if s.IntervalDays != MinIntervalDays {
		t.Errorf("IntervalDays = %v, want %v", s.IntervalDays, MinIntervalDays)
	}
	wantDue := epoch.Add(time.Duration(RelearningSteps[0]) * time.Minute)
	if !s.Due.Equal(wantDue) {
		t.Errorf("Due = %v, want %v", s.Due, wantDue)
	}
}

// Ease can never be driven below MinEase, no matter how many lapses stack up.
func TestAnswer_EaseFloorsAtMinEase(t *testing.T) {
	s := State{Stage: StageReview, EaseFactor: MinEase + 0.05, IntervalDays: 10, Due: epoch}
	s = Answer(s, false, epoch)

	if s.EaseFactor != MinEase {
		t.Errorf("EaseFactor = %v, want floor %v", s.EaseFactor, MinEase)
	}

	// Answering again from the floor must not push it lower.
	s.Stage = StageReview
	s = Answer(s, false, epoch)
	if s.EaseFactor != MinEase {
		t.Errorf("EaseFactor after second lapse = %v, want floor %v", s.EaseFactor, MinEase)
	}
}

// Relearning must re-graduate through its own step list (not the learning
// steps) and restore the interval the lapse pre-assigned, rather than
// resetting to the fresh-card graduating interval.
func TestAnswer_RelearningGraduatesBackToReviewWithLapsedInterval(t *testing.T) {
	s := State{Stage: StageRelearning, StepIndex: 0, EaseFactor: 2.3, IntervalDays: MinIntervalDays, Due: epoch}

	for i, step := range RelearningSteps {
		s = Answer(s, true, epoch)
		if i < len(RelearningSteps)-1 {
			if s.Stage != StageRelearning {
				t.Fatalf("mid relearning: Stage = %q, want %q", s.Stage, StageRelearning)
			}
		} else {
			if s.Stage != StageReview {
				t.Fatalf("after final relearning step: Stage = %q, want %q", s.Stage, StageReview)
			}
			if s.IntervalDays != MinIntervalDays {
				t.Errorf("IntervalDays = %v, want preserved lapse interval %v", s.IntervalDays, MinIntervalDays)
			}
			_ = step
		}
	}
}

func TestMastered(t *testing.T) {
	cases := []struct {
		name string
		s    State
		want bool
	}{
		{"new card", State{Stage: StageNew}, false},
		{"learning card", State{Stage: StageLearning}, false},
		{"review below threshold", State{Stage: StageReview, IntervalDays: MasteredIntervalDays - 1}, false},
		{"review at threshold", State{Stage: StageReview, IntervalDays: MasteredIntervalDays}, true},
		{"review above threshold", State{Stage: StageReview, IntervalDays: MasteredIntervalDays + 10}, true},
		{"relearning above threshold interval", State{Stage: StageRelearning, IntervalDays: MasteredIntervalDays + 10}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.Mastered(); got != tc.want {
				t.Errorf("Mastered() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsDue(t *testing.T) {
	s := State{Due: epoch}
	if !s.IsDue(epoch) {
		t.Error("IsDue(epoch) = false, want true (due exactly now)")
	}
	if !s.IsDue(epoch.Add(time.Minute)) {
		t.Error("IsDue(after due) = false, want true")
	}
	if s.IsDue(epoch.Add(-time.Minute)) {
		t.Error("IsDue(before due) = true, want false")
	}
}
