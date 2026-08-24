package alicercelabs

import "context"

// CronService is the client's scheduled-jobs API — client.Cron.
type CronService struct{ c *Client }

// CronJob is a scheduled job — the exact field set accepted by
// Create/Update mirrors the Cron docs page (name, schedule, image,
// command, and a few more); returned as a generic map since job
// definitions vary in shape by run type.
type CronJob map[string]any

// Create creates a scheduled job.
func (s *CronService) Create(ctx context.Context, job CronJob) (CronJob, error) {
	body, err := jsonBody(job)
	if err != nil {
		return nil, err
	}
	var out CronJob
	if err := s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/cron/jobs", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// List returns every scheduled job for this client.
func (s *CronService) List(ctx context.Context) ([]CronJob, error) {
	var out []CronJob
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/cron/jobs", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get returns one job by id.
func (s *CronService) Get(ctx context.Context, id string) (CronJob, error) {
	var out CronJob
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/cron/jobs/"+id, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Update changes a job's fields.
func (s *CronService) Update(ctx context.Context, id string, fields CronJob) (CronJob, error) {
	body, err := jsonBody(fields)
	if err != nil {
		return nil, err
	}
	var out CronJob
	if err := s.c.doJSON(ctx, "PUT", s.c.APIBase, "/api/v1/cron/jobs/"+id, nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Delete removes a job.
func (s *CronService) Delete(ctx context.Context, id string) error {
	return s.c.doJSON(ctx, "DELETE", s.c.APIBase, "/api/v1/cron/jobs/"+id, nil, nil, nil)
}

// Trigger runs a job right now, outside its schedule.
func (s *CronService) Trigger(ctx context.Context, id string) error {
	return s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/cron/jobs/"+id+"/trigger", nil, nil, nil)
}

// WorkerStatus reports whether the cron daemon is running.
func (s *CronService) WorkerStatus(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/cron/worker/status", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// WorkerStart starts the cron daemon.
func (s *CronService) WorkerStart(ctx context.Context) error {
	return s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/cron/worker/start", nil, nil, nil)
}

// WorkerStop stops the cron daemon.
func (s *CronService) WorkerStop(ctx context.Context) error {
	return s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/cron/worker/stop", nil, nil, nil)
}

// UpTimeService is the client's uptime-monitoring API — client.UpTime.
type UpTimeService struct{ c *Client }

// UpTimeMonitor is a monitored URL.
type UpTimeMonitor map[string]any

// Create creates a monitor. fields can include method, expected_status,
// interval_sec, timeout_sec.
func (s *UpTimeService) Create(ctx context.Context, urlToMonitor string, fields UpTimeMonitor) (UpTimeMonitor, error) {
	payload := UpTimeMonitor{"url": urlToMonitor}
	for k, v := range fields {
		payload[k] = v
	}
	body, err := jsonBody(payload)
	if err != nil {
		return nil, err
	}
	var out UpTimeMonitor
	if err := s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/uptime/monitors", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// List returns every monitor for this client.
func (s *UpTimeService) List(ctx context.Context) ([]UpTimeMonitor, error) {
	var out []UpTimeMonitor
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/uptime/monitors", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get returns one monitor by id.
func (s *UpTimeService) Get(ctx context.Context, id string) (UpTimeMonitor, error) {
	var out UpTimeMonitor
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/uptime/monitors/"+id, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Update changes a monitor's fields.
func (s *UpTimeService) Update(ctx context.Context, id string, fields UpTimeMonitor) (UpTimeMonitor, error) {
	body, err := jsonBody(fields)
	if err != nil {
		return nil, err
	}
	var out UpTimeMonitor
	if err := s.c.doJSON(ctx, "PUT", s.c.APIBase, "/api/v1/uptime/monitors/"+id, nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Delete removes a monitor.
func (s *UpTimeService) Delete(ctx context.Context, id string) error {
	return s.c.doJSON(ctx, "DELETE", s.c.APIBase, "/api/v1/uptime/monitors/"+id, nil, nil, nil)
}

// Checks returns the check history for one monitor.
func (s *UpTimeService) Checks(ctx context.Context, id string) ([]map[string]any, error) {
	var out []map[string]any
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/uptime/monitors/"+id+"/checks", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// WorkerStatus reports whether the uptime daemon is running.
func (s *UpTimeService) WorkerStatus(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/uptime/worker/status", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// WorkerStart starts the uptime daemon.
func (s *UpTimeService) WorkerStart(ctx context.Context) error {
	return s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/uptime/worker/start", nil, nil, nil)
}

// WorkerStop stops the uptime daemon.
func (s *UpTimeService) WorkerStop(ctx context.Context) error {
	return s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/uptime/worker/stop", nil, nil, nil)
}
