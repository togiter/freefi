package utils

import "time"

func TimeFmt(t int64, fmtStr string) string {
	if fmtStr == "" {
		fmtStr = "2006-01-02 15:04:05"
	}
	return time.Unix(t, 0).Format(fmtStr)
}
