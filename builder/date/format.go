package format

import (
	"strconv"
	"strings"
	"time"
)

func Duration(d time.Duration) string {
	// taken from github.com/hako/durafmt
	var (
		units = []string{"days", "hours", "minutes", "seconds"}
	)

	var duration string
	input := d.String()

	// Convert duration.
	seconds := int(d.Seconds()) % 60
	minutes := int(d.Minutes()) % 60
	hours := int(d.Hours()) % 24
	days := int(d/(24*time.Hour)) % 365 % 7
	// Create a map of the converted duration time.
	durationMap := map[string]int{
		"seconds": seconds,
		"minutes": minutes,
		"hours":   hours,
		"days":    days,
	}

	// Construct duration string.
	for _, u := range units {
		v := durationMap[u]
		strval := strconv.Itoa(v)
		switch {
		// add to the duration string if v > 1.
		case v > 1:
			duration += strval + " " + u + " "
			// remove the plural 's', if v is 1.
		case v == 1:
			duration += strval + " " + strings.TrimRight(u, "s") + " "
			// omit any value with 0s or 0.
		case d.String() == "0" || d.String() == "0s":
			// note: milliseconds and minutes have the same suffix (m)
			// so we have to check if the units match with the suffix.

			// check for a suffix that is NOT the milliseconds suffix.
			if strings.HasSuffix(input, string(u[0])) && !strings.Contains(input, "ms") {
				// if it happens that the units are milliseconds, skip.
				if u == "milliseconds" {
					continue
				}
				duration += strval + " " + u
			}
			break
			// omit any value with 0.
		case v == 0:
			continue
		}
	}
	// trim any remaining spaces.
	duration = strings.TrimSpace(duration)
	return duration
}
