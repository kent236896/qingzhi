export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
  requestId: string
}

export interface HealthLive {
  status: 'live'
}

export interface HealthChecks {
  database: 'up' | 'down'
  redis: 'up' | 'down'
}

export interface HealthReady {
  status: 'ready' | 'not_ready'
  checks: HealthChecks
}
