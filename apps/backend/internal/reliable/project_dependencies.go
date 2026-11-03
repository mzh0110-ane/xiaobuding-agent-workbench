package reliable

import (
	"encoding/json"
	"errors"

	bolt "go.etcd.io/bbolt"
)

func (e *Engine) checkProjectDependencies(project, itemID string) error {
	return e.Store.db.View(func(tx *bolt.Tx) error {
		record, err := readProject(tx, project)
		if err != nil {
			return err
		}
		index := projectItemIndex(record, itemID)
		if index < 0 {
			return errors.New("project_item_unavailable: the requirement no longer exists")
		}
		return projectDependenciesAccepted(tx, record, record.Items[index])
	})
}

func projectDependenciesAccepted(tx *bolt.Tx, record *ProjectRecord, item ProjectItem) error {
	for _, id := range item.DependsOn {
		index := projectItemIndex(record, id)
		if index < 0 {
			return errors.New("project_dependency_pending: a required backlog item is missing")
		}
		dependency := record.Items[index]
		if dependency.TaskID == "" {
			return errors.New("project_dependency_pending: a dependency has not been accepted by the user")
		}
		var task Task
		if err := json.Unmarshal(tx.Bucket(tasksBucket).Get([]byte(dependency.TaskID)), &task); err != nil {
			return err
		}
		if task.State != stateAccepted || projectKey(task.Project) != projectKey(record.Project) {
			return errors.New("project_dependency_pending: a dependency needs human acceptance before dispatch")
		}
	}
	return nil
}
