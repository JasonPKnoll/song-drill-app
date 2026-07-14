// Package srs implements an Anki-style spaced-repetition scheduler: new
// cards move through same-day "learning" steps (minutes-scale), graduate
// into day-scale "review" intervals governed by a per-card ease factor,
// and a miss in review drops the card into "relearning" (same-day steps
// again) before it re-graduates. This mirrors Anki's classic SM-2-derived
// scheduler (not FSRS) at the level of a plain correct/incorrect grade —
// there's no Hard/Easy distinction, so ease only ever moves down (on a
// lapse) or stays flat (on a pass); it never recovers upward the way a
// real Anki "Easy" press would. That's a deliberate simplification: Anki's
// own community broadly advises sticking to Again/Good and skipping
// Hard/Easy, since they're hard to grade consistently and mostly add noise.
//
// This package is deliberately self-contained: no imports from the rest of
// song-drill, no domain concepts (no "vocab" or "line", just an opaque
// card's review state). That's so it can be lifted into a separate module
// later with no rework, once a second real application actually needs it —
// see the "should this be a standalone library" conversation this was born
// from. Building it prematurely as its own module/repo before there's a
// second consumer risks guessing the wrong abstraction; keeping it
// dependency-free here costs nothing and preserves that option.
package srs

import "time"

type Stage string

const (
	StageNew        Stage = "new"
	StageLearning   Stage = "learning"
	StageReview     Stage = "review"
	StageRelearning Stage = "relearning"
)

// Learning/relearning steps, in minutes — Anki's own defaults ("1m 10m"
// learning steps, "10m" relearning step).
var (
	LearningSteps   = []int{1, 10}
	RelearningSteps = []int{10}
)

const (
	GraduatingIntervalDays = 1.0  // first review-state interval, once learning steps are passed
	StartingEase           = 2.5  // Anki default (250%)
	MinEase                = 1.3  // Anki never lets ease drop below 130%
	EaseAgainPenalty       = 0.20 // a lapse drops ease by 20 percentage points
	MinIntervalDays        = 1.0  // Anki's default "new interval" after a lapse resets to this
	LeechThreshold         = 8    // lapses at which Anki flags a card as a leech (tracked, not acted on)
	MasteredIntervalDays   = 30.0 // display/stats threshold for "mastered", not part of the algorithm itself
	BurnedIntervalDays     = 3650 // ~10 years — long enough a manually-burned card won't resurface in practice
)

// State is a single card's full review state, independent of whatever
// storage or content it's attached to.
type State struct {
	Stage        Stage
	StepIndex    int // position within the current learning/relearning steps
	EaseFactor   float64
	IntervalDays float64 // last computed review-state interval
	Lapses       int     // times this card has been missed while in the review stage
	Due          time.Time
}

// New returns the initial state for a card that has never been studied.
func New(now time.Time) State {
	return State{Stage: StageNew, EaseFactor: StartingEase, Due: now}
}

// Burned returns the state for a card the learner has manually flagged as
// already known — a stats-sheet override, not something the scheduler
// itself ever produces from an answer. It's just a review-stage card with
// an interval far past the mastered threshold, so it's indistinguishable
// from a card that was naturally drilled to that point: it counts as
// mastered, stays out of the drill queue, and would resume completely
// normal ease-based scheduling if it were ever answered again.
func Burned(now time.Time) State {
	return State{
		Stage:        StageReview,
		EaseFactor:   StartingEase,
		IntervalDays: BurnedIntervalDays,
		Due:          now.Add(daysToDuration(BurnedIntervalDays)),
	}
}

// IsDue reports whether a card should be shown for review at the given time.
func (s State) IsDue(now time.Time) bool {
	return !s.Due.After(now)
}

// Mastered reports whether a card's review interval has grown past the
// "practically memorized" threshold — a display/stats concept, not
// something the scheduling algorithm itself needs.
func (s State) Mastered() bool {
	return s.Stage == StageReview && s.IntervalDays >= MasteredIntervalDays
}

// Answer applies a single correct/incorrect grade to a card's state,
// returning its new state. now is the moment of the review.
func Answer(s State, correct bool, now time.Time) State {
	if s.Stage == StageReview {
		return answerReview(s, correct, now)
	}

	// New, Learning, and Relearning cards all move through the same
	// step-based mechanic — only which step list applies differs.
	stage, steps := StageLearning, LearningSteps
	if s.Stage == StageRelearning {
		stage, steps = StageRelearning, RelearningSteps
	}
	return answerSteps(s, stage, steps, correct, now)
}

func answerSteps(s State, stage Stage, steps []int, correct bool, now time.Time) State {
	if !correct {
		// A miss during learning/relearning sends the card back to the very
		// first step — same-day, see it again soon — not out of the phase
		// entirely. This is the "resets progress for that word for the day"
		// behavior.
		s.Stage = stage
		s.StepIndex = 0
		s.Due = now.Add(time.Duration(steps[0]) * time.Minute)
		return s
	}

	s.StepIndex++
	if s.StepIndex >= len(steps) {
		// Passed every step — graduate into (or back into) the review stage.
		// A fresh card gets the standard graduating interval; a card
		// graduating back out of relearning instead keeps the interval the
		// lapse already assigned it in answerReview.
		s.Stage = StageReview
		s.StepIndex = 0
		if s.IntervalDays <= 0 {
			s.IntervalDays = GraduatingIntervalDays
		}
		s.Due = now.Add(daysToDuration(s.IntervalDays))
		return s
	}

	s.Stage = stage
	s.Due = now.Add(time.Duration(steps[s.StepIndex-1]) * time.Minute)
	return s
}

func answerReview(s State, correct bool, now time.Time) State {
	if correct {
		s.IntervalDays *= s.EaseFactor
		s.Due = now.Add(daysToDuration(s.IntervalDays))
		return s
	}

	// Lapse: penalize ease, drop into relearning, and pre-assign the
	// interval this card will carry once relearning finishes — Anki's
	// default "new interval" on a lapse is the minimum interval, i.e. the
	// long-earned interval is forfeited, not just discounted.
	s.Lapses++
	s.EaseFactor = maxFloat(MinEase, s.EaseFactor-EaseAgainPenalty)
	s.IntervalDays = MinIntervalDays
	s.Stage = StageRelearning
	s.StepIndex = 0
	s.Due = now.Add(time.Duration(RelearningSteps[0]) * time.Minute)
	return s
}

func daysToDuration(days float64) time.Duration {
	return time.Duration(days * float64(24*time.Hour))
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
