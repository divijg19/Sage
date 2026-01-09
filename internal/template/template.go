package template

type Template struct {
	Name          string
	SuggestedKind string // "record" | "decision" | ""
	Body          string
}
