package fingers

// EnableMatchDetail enables matcher detail collection on the underlying fingers
// engine. After enabling it, match results expose matcher metadata through
// Framework.MatchDetail.
func (e *Engine) EnableMatchDetail() error {
	if e == nil || e.engine == nil {
		return nil
	}
	fEngine, err := e.GetFingersEngine()
	if err != nil {
		return err
	}
	if fEngine == nil {
		return nil
	}
	fEngine.EnableMatchDetail()
	return nil
}
