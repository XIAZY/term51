// Tests translated one for one from src/terminal/ScreenSet.zig, the tests
// under its banner `term51: additional tests (AA)` (Ghostty v1.3.1 plus the
// term51 additional tests). Each Go test carries the exact Zig title in a
// "// Zig: test" comment. Helpers are prefixed sca; see Screen_5_test.go for
// the translation notes (they apply here too). Tests were not translated: none.

package terminal

import "testing"

// ---------------------------------------------------------------------------
// ScreenSet.zig: term51: additional tests (AA).
// ---------------------------------------------------------------------------

// scaSetOpts is term51AAOpts: `.{ .cols = 10, .rows = 5, .max_scrollback =
// 0 }` with the Zig default kitty_image_storage_limit.
func scaSetOpts() ScreenOptions {
	return ScreenOptions{
		Cols:                   10,
		Rows:                   5,
		MaxScrollback:          0,
		KittyImageStorageLimit: 320 * 1000 * 1000,
	}
}

// scaSetInit is `try ScreenSet.init(alloc, opts)`; the set is returned by
// pointer (the methods take a pointer receiver).
func scaSetInit(t *testing.T, opts ScreenOptions) *ScreenSet {
	t.Helper()
	set, err := ScreenSetInit(opts)
	zigTry(t, err)
	return &set
}

// scaSetGetInit is `try set.getInit(alloc, key, opts)`.
func scaSetGetInit(t *testing.T, set *ScreenSet, key ScreenSetKey, opts ScreenOptions) *Screen {
	t.Helper()
	s, err := set.GetInit(key, opts)
	zigTry(t, err)
	return s
}

// Zig: test "term51: get: the lookup does not depend on the active screen"
func TestScaSet_get_the_lookup_does_not_depend_on_the_active_screen(t *testing.T) {
	testStdCapacity(t)
	set := scaSetInit(t, scaSetOpts())
	defer set.Deinit()
	primary := set.Active
	alt := scaSetGetInit(t, set, ScreenSetKeyAlternate, scaSetOpts())

	// Primary is active.
	expect(t, set.Get(ScreenSetKeyPrimary) == primary, "set.get(.primary) == primary")
	expect(t, set.Get(ScreenSetKeyAlternate) == alt, "set.get(.alternate) == alt")
	expectEqual[ScreenSetKey](t, ScreenSetKeyPrimary, set.ActiveKey)
	expect(t, set.Active == primary, "set.active == primary")

	// Alternate is active: both are still found and nothing is switched.
	set.SwitchTo(ScreenSetKeyAlternate)
	expect(t, set.Get(ScreenSetKeyPrimary) == primary, "set.get(.primary) == primary")
	expect(t, set.Get(ScreenSetKeyAlternate) == alt, "set.get(.alternate) == alt")
	expectEqual[ScreenSetKey](t, ScreenSetKeyAlternate, set.ActiveKey)
	expect(t, set.Active == alt, "set.active == alt")
}

// Zig: test "term51: get: every key, repeated lookups and the key values"
func TestScaSet_get_every_key_repeated_lookups_and_the_key_values(t *testing.T) {
	testStdCapacity(t)
	set := scaSetInit(t, scaSetOpts())
	defer set.Deinit()

	expectEqual[ScreenSetKey](t, 0, ScreenSetKeyPrimary)
	expectEqual[ScreenSetKey](t, 1, ScreenSetKeyAlternate)
	// Go: the number of enum fields of Key is the number of slots of the
	// EnumMap array.
	expectEqual[uint](t, 2, uint(len(set.all.values)))

	keys := []ScreenSetKey{ScreenSetKeyPrimary, ScreenSetKeyAlternate}

	// Only the primary screen exists, however often it is asked for.
	for _, key := range keys {
		want_found := key == ScreenSetKeyPrimary
		for i := 0; i < 3; i++ {
			expectEqual[bool](t, want_found, set.Get(key) != nil)
		}
	}
	expect(t, set.Get(ScreenSetKeyPrimary) == set.Get(ScreenSetKeyPrimary),
		"set.get(.primary) == set.get(.primary)")

	alt := scaSetGetInit(t, set, ScreenSetKeyAlternate, scaSetOpts())
	for _, key := range keys {
		expect(t, set.Get(key) != nil, "set.get(key) != null")
		expect(t, set.Get(key) == set.Get(key), "set.get(key) == set.get(key)")
	}
	expect(t, set.Get(ScreenSetKeyAlternate) == alt, "set.get(.alternate) == alt")
	expect(t, set.Get(ScreenSetKeyPrimary) != set.Get(ScreenSetKeyAlternate),
		"set.get(.primary) != set.get(.alternate)")
}

// Zig: test "term51: get: returns the live screen, not a copy"
func TestScaSet_get_returns_the_live_screen_not_a_copy(t *testing.T) {
	testStdCapacity(t)
	set := scaSetInit(t, scaSetOpts())
	defer set.Deinit()
	scaSetGetInit(t, set, ScreenSetKeyAlternate, scaSetOpts())

	alt := set.Get(ScreenSetKeyAlternate)
	expect(t, alt != nil, "set.get(.alternate) != null")
	alt.CursorAbsolute(3, 2)
	expectEqual[uint](t, 3, uint(set.Get(ScreenSetKeyAlternate).Cursor.X))
	expectEqual[uint](t, 2, uint(set.Get(ScreenSetKeyAlternate).Cursor.Y))
	// The other screen is not affected.
	expectEqual[uint](t, 0, uint(set.Get(ScreenSetKeyPrimary).Cursor.X))
	expectEqual[uint](t, 0, uint(set.Get(ScreenSetKeyPrimary).Cursor.Y))

	set.SwitchTo(ScreenSetKeyAlternate)
	expectEqual[uint](t, 3, uint(set.Active.Cursor.X))
}

// Zig: test "term51: get: a removed screen is gone and a re-created one is found"
func TestScaSet_get_a_removed_screen_is_gone_and_a_re_created_one_is_found(t *testing.T) {
	testStdCapacity(t)
	set := scaSetInit(t, scaSetOpts())
	defer set.Deinit()

	scaSetGetInit(t, set, ScreenSetKeyAlternate, scaSetOpts())
	set.Remove(ScreenSetKeyAlternate)
	expect(t, set.Get(ScreenSetKeyAlternate) == nil, "set.get(.alternate) == null")
	expect(t, set.Get(ScreenSetKeyPrimary) != nil, "set.get(.primary) != null")

	again := scaSetGetInit(t, set, ScreenSetKeyAlternate, ScreenOptions{
		Cols:                   3,
		Rows:                   2,
		MaxScrollback:          0,
		KittyImageStorageLimit: 320 * 1000 * 1000,
	})
	expect(t, set.Get(ScreenSetKeyAlternate) == again, "set.get(.alternate) == again")
	expectEqual[uint](t, 3, uint(set.Get(ScreenSetKeyAlternate).Pages.cols))
	expectEqual[uint](t, 2, uint(set.Get(ScreenSetKeyAlternate).Pages.rows))
}
