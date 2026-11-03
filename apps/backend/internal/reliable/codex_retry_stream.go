package reliable

import (
	"regexp"
	"strconv"
	"strings"
)

var codexReconnectNoticePattern = regexp.MustCompile(`^Reconnecting\.\.\. ([1-9][0-9]{0,2})/([1-9][0-9]{0,2}) \(([^\r\n]+)\)$`)

func isCodexReconnectNotice(message string) bool {
	match := codexReconnectNoticePattern.FindStringSubmatch(message)
	if match == nil || strings.TrimSpace(match[3]) == "" {
		return false
	}
	// The bounded decimal captures always fit in an int.
	attempt, _ := strconv.Atoi(match[1])
	limit, _ := strconv.Atoi(match[2])
	return attempt <= limit
}
