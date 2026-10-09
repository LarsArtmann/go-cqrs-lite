package id

import errorfamily "github.com/larsartmann/go-error-family"

var ErrEmptyString error = errorfamily.NewRejection("id.empty_string", "empty string")
