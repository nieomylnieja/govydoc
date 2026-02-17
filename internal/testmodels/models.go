package testmodels

import (
	"fmt"

	"github.com/nieomylnieja/govydoc/internal/testmodels/moremodels"
)

// Teacher is a sample struct used for testing.
// Spoiler alert: it has [Student].
// [Student.Name] is the name of the student.
//
// Teacher attends [moremodels.University].
type Teacher struct {
	// Name is the name of the teacher.
	Name  string `json:"name"`
	Hobby string `json:"hobby"`
	Age   int    `json:"age"` // Age is the age of the teacher. Note: This is no ta valid doc string.
	// Students is a list of students.
	Students   []Student             `json:"students"`
	University moremodels.University `json:"university"` // University is the university of the teacher.
	NoTag      int
	Stringer   fmt.Stringer `json:"stringer"`
}

// Student is just a teacher!
// You must see [fmt.Stringer] though.
// Don't forget to visit [this site].
// Have you seen [Teacher]?
//
// Deprecated: Use Teacher instead.
//
// [this site]: https://example.com
type Student struct {
	// Age is life!
	Age int `json:"age"`
	// Some comment.
	Name string `json:"name"`
	// Deprecated: Use Name instead.
	OldName string `json:"oldName"`
}
