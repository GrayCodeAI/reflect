package config

// DeploymentRoutingEnabled delegates deployment-routing policy ownership to Relay runtime.
func DeploymentRoutingEnabled(s Settings) bool {
	engine, err := newRelayEngine()
	return err == nil && engine.DeploymentRoutingEnabled(s.DeploymentRouting)
}
