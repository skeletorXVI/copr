package core

// Update is the change a package is about to publish: the version currently
// packaged, the release replacing it, and the RPM release number the result
// will carry. From is empty when a package is pinned for the first time, and
// equals To's version when only the packaging changed.
type Update struct {
	Name    string
	From    string
	To      Release
	Release int
}

// Action is how an update reaches main. Every update that is published at all
// goes through a pull request, so the only decision left to a policy is
// whether to publish it.
type Action string

const (
	// Review opens or updates a pull request.
	Review Action = "review"
	// Skip publishes nothing. The package stays on its current version.
	Skip Action = "skip"
)

// Policy decides what should happen to an update. It replaces a fixed set of
// modes: a package that needs a rule of its own writes one, without the core
// having to anticipate it.
type Policy interface {
	Decide(Update) Action
}

// PolicyFunc adapts a plain function to Policy.
type PolicyFunc func(Update) Action

func (f PolicyFunc) Decide(u Update) Action { return f(u) }

// ReviewAll sends every update through a pull request.
type ReviewAll struct{}

func (ReviewAll) Decide(Update) Action { return Review }
