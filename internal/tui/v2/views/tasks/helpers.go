package tasks

func (m Model) selectedTaskName() string {
	if len(m.Tasks) == 0 || m.Selected >= len(m.Tasks) {
		return ""
	}
	return m.Tasks[m.Selected].Name
}
