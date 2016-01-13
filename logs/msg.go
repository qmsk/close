package logs

type LogMsg struct {
    Line        string      `json:"line"`

    // stats
    Dropped     uint        `json:"dropped,omitempty"`
}
