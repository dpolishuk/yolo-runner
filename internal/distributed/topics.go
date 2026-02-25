package distributed

type EventSubjects struct {
	Register       string
	Heartbeat      string
	TaskDispatch   string
	TaskResult     string
	ServiceRequest string
	ServiceResult  string
}

func DefaultEventSubjects(prefix string) EventSubjects {
	if prefix == "" {
		prefix = "yolo"
	}
	return EventSubjects{
		Register:       prefix + ".executor.register",
		Heartbeat:      prefix + ".executor.heartbeat",
		TaskDispatch:   prefix + ".task.dispatch",
		TaskResult:     prefix + ".task.result",
		ServiceRequest: prefix + ".service.request",
		ServiceResult:  prefix + ".service.response",
	}
}
