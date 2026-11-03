package reliable

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAcceptanceRejectsMissingOrAlteredEvidence(t *testing.T) {
	for _, kind := range []string{"missing-log", "altered-log", "altered-review", "missing-source", "missing-contract"} {
		t.Run(kind, func(t *testing.T) {
			e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
			if err := e.RunTask(context.Background(), task.ID); err != nil {
				t.Fatal(err)
			}
			root := filepath.Join(e.Root, "evidence", task.ID)
			var err error
			switch kind {
			case "missing-log":
				err = os.Remove(filepath.Join(root, "round-1", "check-1.log"))
			case "altered-log":
				err = os.WriteFile(filepath.Join(root, "round-1", "check-1.log"), []byte("forged passing output"), 0600)
			case "altered-review":
				err = os.WriteFile(filepath.Join(root, "round-1-review.json"), []byte(`{"verdict":"rework"}`), 0600)
			case "missing-source":
				err = os.Remove(filepath.Join(root, "round-1-source.json"))
			case "missing-contract":
				err = os.Remove(filepath.Join(root, "contract.json"))
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = e.Accept(context.Background(), task.ID); err == nil {
				t.Fatal("candidate was applied without intact evidence")
			}
			original, err := os.ReadFile(filepath.Join(task.Project, "app.txt"))
			if err != nil || string(original) != "before\n" {
				t.Fatalf("invalid evidence reached the original project: %s %v", original, err)
			}
		})
	}
}
