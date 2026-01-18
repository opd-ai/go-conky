package conky

import "time"

// HealthStatus represents the overall health state of a component.
type HealthStatus string

const (
	// HealthOK indicates the component is functioning normally.
	HealthOK HealthStatus = "ok"
	// HealthDegraded indicates partial functionality or non-critical issues.
	HealthDegraded HealthStatus = "degraded"
	// HealthUnhealthy indicates the component is not functioning.
	HealthUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck contains the health status of the Conky instance and its components.
type HealthCheck struct {
	// Status is the overall health status.
	Status HealthStatus

	// Timestamp is when the health check was performed.
	Timestamp time.Time

	// Uptime is the duration since the instance started (zero if not running).
	Uptime time.Duration

	// Components contains health status for individual components.
	Components map[string]ComponentHealth

	// Message provides additional context about the health status.
	Message string
}

// ComponentHealth represents the health status of an individual component.
type ComponentHealth struct {
	// Status is the health status of this component.
	Status HealthStatus

	// Message provides details about the component's state.
	Message string

	// LastUpdated is when this component was last successfully updated.
	LastUpdated time.Time
}

// IsHealthy returns true if the overall status is HealthOK.
func (h HealthCheck) IsHealthy() bool {
	return h.Status == HealthOK
}

// IsDegraded returns true if the overall status is HealthDegraded.
func (h HealthCheck) IsDegraded() bool {
	return h.Status == HealthDegraded
}

// IsUnhealthy returns true if the overall status is HealthUnhealthy.
func (h HealthCheck) IsUnhealthy() bool {
	return h.Status == HealthUnhealthy
}
