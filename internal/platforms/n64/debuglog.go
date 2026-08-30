package n64

import (
	"encoding/json"
	"os"
	"time"
)

const debugLogPath = `D:\projects\ps1_blazium\debug-098750.log`

// #region agent log
func agentLog(hypothesisID, location, message string, data map[string]any) {
	f, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	payload := map[string]any{
		"sessionId":    "098750",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
		"runId":        "n64-impl",
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
}

// #endregion
