package catalog

type Name string

func (n Name) String() string { return string(n) }

func (n Name) IsZero() bool { return n == "" }

// cqrs-lint:ignore(A008) library code or intentional pattern

type Version string

func (v Version) String() string { return string(v) }

func (v Version) IsZero() bool { return v == "" }

type Summary string

func (s Summary) String() string { return string(s) }

func (s Summary) IsZero() bool { return s == "" }

type Title string

func (t Title) String() string { return string(t) }

func (t Title) IsZero() bool { return t == "" }

type Description string

func (d Description) String() string { return string(d) }

func (d Description) IsZero() bool { return d == "" }

type Address string

func (a Address) String() string { return string(a) }

func (a Address) IsZero() bool { return a == "" }

type Protocol string

func (p Protocol) String() string { return string(p) }

func (p Protocol) IsZero() bool { return p == "" }

type Host string

func (h Host) String() string { return string(h) }

func (h Host) IsZero() bool { return h == "" }

type Email string

func (e Email) String() string { return string(e) }

func (e Email) IsZero() bool { return e == "" }

type URL string

func (u URL) String() string { return string(u) }

func (u URL) IsZero() bool { return u == "" }

type ContentType string

func (c ContentType) String() string { return string(c) }

func (c ContentType) IsZero() bool { return c == "" }

type DeliveryGuarantee string

func (d DeliveryGuarantee) String() string { return string(d) }

func (d DeliveryGuarantee) IsZero() bool { return d == "" }

type Method string

func (m Method) String() string { return string(m) }

func (m Method) IsZero() bool { return m == "" }

type Icon string

func (i Icon) String() string { return string(i) }

func (i Icon) IsZero() bool { return i == "" }

type Color string

func (c Color) String() string { return string(c) }

func (c Color) IsZero() bool { return c == "" }

type Language string

func (l Language) String() string { return string(l) }

func (l Language) IsZero() bool { return l == "" }

type Role string

func (r Role) String() string { return string(r) }

func (r Role) IsZero() bool { return r == "" }
