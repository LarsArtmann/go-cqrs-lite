package id

import errorfamily "github.com/larsartmann/go-error-family"

var errEmptyString = errorfamily.NewRejection("id.empty_string", "empty string")
