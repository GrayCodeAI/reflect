package config

// DeploymentRoutingEnabled delegates deployment-routing policy ownership to Mux runtime.
func DeploymentRoutingEnabled(s Settings) bool {
	engine, err := newMuxEngine()
	return err == nil && engine.DeploymentRoutingEnabled(s.DeploymentRouting)
}
