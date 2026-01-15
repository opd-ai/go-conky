//go:build noebiten

package conky

// runRenderLoop is a no-op in noebiten builds.
// In headless/noebiten mode, there is no rendering loop.
func (c *conkyImpl) runRenderLoop() {
	// No-op: headless mode uses the wait-for-context-cancel pattern
	// which is handled in the Start() method's else branch
	<-c.ctx.Done()
}
