// Package srs implements the spaced-repetition interval schedule shared by
// vocab and line drills. Same interval table as jp-drill.
package srs

import "time"

var intervals = []int{0, 1, 3, 7, 14, 30, 90}

const MasteredStreak = 5

// NextReview returns the next review date (YYYY-MM-DD) for a given streak.
func NextReview(streak int) string {
	return NextReviewFrom(time.Now(), streak)
}

// NextReviewFrom is NextReview with an explicit reference time, for testing.
func NextReviewFrom(from time.Time, streak int) string {
	days := intervals[min(streak, len(intervals)-1)]
	return from.AddDate(0, 0, days).Format("2006-01-02")
}

// Update applies a drill result to a streak, returning the new streak and
// next review date. Missing a card resets the streak to 0.
func Update(streak int, gotIt bool) (newStreak int, nextReview string) {
	if gotIt {
		newStreak = streak + 1
	} else {
		newStreak = 0
	}
	return newStreak, NextReview(newStreak)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
