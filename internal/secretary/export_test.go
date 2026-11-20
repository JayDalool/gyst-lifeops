package secretary

// SetAfterEnterWait installs a hook that runs in EnsureFresh after a caller has
// registered as waiting and before it starts waiting, so a test can pause a
// caller at that point without racing on sleeps (GYST-164).
func SetAfterEnterWait(f *Freshener, fn func()) { f.afterEnter = fn }
