package utils

import (
	"fmt"
	"regexp"
	"strconv"
)

func AdjustedTimeToKyiv(locationStr string) string {
	re := regexp.MustCompile(`(Сьогодні|Вчора) о (\d{1,2}):(\d{2})`)

	return re.ReplaceAllStringFunc(locationStr, func(match string) string{
		parts := re.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}

		day := parts[1]
		hour, _ := strconv.Atoi(parts[2])
		minute := parts[3]

		hour += 3
		if hour >= 24 {
			hour -= 24
		}

		return fmt.Sprintf("%s о %02d:%s", day, hour, minute)
	})
}