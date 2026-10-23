package keycodec

import "strings"

// The v1 key grammar joins segments with Sep (0x00) without escaping, so a
// segment CONTAINING Sep collides with a different segment split: the pair
// ("a", "b\x00c") and ("a\x00b", "c") produce the identical map key —
// cross-collection injection and silent overwrite (the separator-injection
// bug the v2 family kills). Escape renders a segment free of Sep and of the
// escape byte so every v2 join is unambiguous.
//
// Scheme: 0x00 → 0x01 '0', 0x01 → 0x01 '1'; every other byte passes through.
// Unescape walks the output and reverses exactly those pairs.
//
// Byte ORDER is not preserved across segments containing the special bytes
// (no single-escape scheme can be); v2 ordering only ever gates iteration
// order, never correctness — sort orders and keyset cursors ride user keys
// (UserKeyBytes) or the seq tail, both independent of segment byte order.

// Escape renders a key segment free of Sep and of the escape byte (0x01),
// so joining escaped segments with Sep is injective.
func Escape(s string) string {
	if !strings.ContainsAny(s, "\x00\x01") {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	for i := range len(s) {
		switch s[i] {
		case 0x00:
			b.WriteByte(0x01)
			b.WriteByte('0')
		case 0x01:
			b.WriteByte(0x01)
			b.WriteByte('1')
		default:
			b.WriteByte(s[i])
		}
	}

	return b.String()
}

// Unescape reverses Escape. Malformed escape pairs (a dangling 0x01 or an
// unknown follower byte) pass through literally — Unescape is total, and
// garbage-in-garbage-out beats an error for scan-side use on foreign data.
func Unescape(s string) string {
	if !strings.ContainsRune(s, 0x01) {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if s[i] == 0x01 && i+1 < len(s) {
			switch s[i+1] {
			case '0':
				b.WriteByte(0x00)
				i++
			case '1':
				b.WriteByte(0x01)
				i++
			default:
				b.WriteByte(0x01)
			}

			continue
		}

		b.WriteByte(s[i])
	}

	return b.String()
}
