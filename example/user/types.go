package main

type Email string

func (e Email) String() string { return string(e) }
func (e Email) IsZero() bool   { return e == "" }

type DisplayName string

func (n DisplayName) String() string { return string(n) }
func (n DisplayName) IsZero() bool   { return n == "" }

type Reason string

func (r Reason) String() string { return string(r) }
func (r Reason) IsZero() bool   { return r == "" }
