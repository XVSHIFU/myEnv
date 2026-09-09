package config

// ResolveExistingPath uses the same junction-aware resolution as workspace
// containment checks. It does not create paths or inspect arbitrary ancestors.
func ResolveExistingPath(path string) (string, error) { return resolveWorkspacePath(path) }
