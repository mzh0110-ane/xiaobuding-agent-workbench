package reliable

import "time"

func recoverDispatches(t *Task) {
	for i := range t.Runs {
		d := &t.Runs[i]
		switch d.Status {
		case "preparing":
			d.Status = dispatchNotStarted
		case "running":
			d.Status = "interrupted"
		default:
			continue
		}
		d.EndEvidence = ""
		d.Completed = time.Now().UTC().Format(time.RFC3339Nano)
	}
}
