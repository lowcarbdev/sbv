import { useState, useEffect } from 'react'
import axios from 'axios'
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell, LineChart, Line, Legend
} from 'recharts'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8085/api'

// Color palette
const COLORS = ['#0d6efd', '#198754', '#ffc107', '#dc3545', '#6c757d', '#0dcaf0', '#6610f2', '#d63384']

function Summary({ startDate, endDate }) {
  const [analytics, setAnalytics] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    fetchAnalytics()
  }, [startDate, endDate])

  const fetchAnalytics = async () => {
    setLoading(true)
    setError(null)
    try {
      const params = {}
      if (startDate) params.start = startDate.toISOString()
      if (endDate) params.end = endDate.toISOString()

      const response = await axios.get(`${API_BASE}/analytics`, { params })
      setAnalytics(response.data)
    } catch (err) {
      console.error('Error fetching analytics:', err)
      setError('Failed to load analytics')
    } finally {
      setLoading(false)
    }
  }

  const formatDuration = (seconds) => {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    if (hours > 0) return `${hours}h ${minutes}m`
    return `${minutes}m`
  }

  const formatHour = (hour) => {
    if (hour === 0) return '12 AM'
    if (hour === 12) return '12 PM'
    return hour < 12 ? `${hour} AM` : `${hour - 12} PM`
  }

  const formatPhoneNumber = (phone) => {
    if (!phone) return ''
    // Remove all non-digit characters
    const digits = phone.replace(/\D/g, '')
    // Format as (XXX) XXX-XXXX if 10 digits, or +X (XXX) XXX-XXXX if 11
    if (digits.length === 10) {
      return `(${digits.slice(0, 3)}) ${digits.slice(3, 6)}-${digits.slice(6)}`
    } else if (digits.length === 11 && digits[0] === '1') {
      return `+1 (${digits.slice(1, 4)}) ${digits.slice(4, 7)}-${digits.slice(7)}`
    }
    return phone
  }

  if (loading) {
    return (
      <div className="h-100 d-flex align-items-center justify-content-center">
        <div className="text-center">
          <div className="spinner-border text-primary mb-3" role="status">
            <span className="visually-hidden">Loading...</span>
          </div>
          <p className="text-muted">Loading analytics...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="h-100 d-flex align-items-center justify-content-center">
        <div className="text-center text-danger">
          <p>{error}</p>
          <button className="btn btn-primary" onClick={fetchAnalytics}>
            Retry
          </button>
        </div>
      </div>
    )
  }

  if (!analytics) return null

  // Prepare data for message type pie chart
  const messageTypeData = [
    { name: 'Sent', value: analytics.total_sent },
    { name: 'Received', value: analytics.total_received }
  ].filter(d => d.value > 0)

  // Prepare data for call type pie chart
  const callTypeData = [
    { name: 'Incoming', value: analytics.incoming_calls },
    { name: 'Outgoing', value: analytics.outgoing_calls },
    { name: 'Missed', value: analytics.missed_calls }
  ].filter(d => d.value > 0)

  // Prepare top contacts data with display names
  const topContactsData = (analytics.top_contacts || []).slice(0, 8).map(c => ({
    ...c,
    displayName: c.contact_name || formatPhoneNumber(c.address) || c.address
  }))

  return (
    <div className="h-100 d-flex flex-column">
      <div className="bg-light border-bottom p-3">
        <h2 className="h5 mb-0 d-flex align-items-center gap-2">
          <svg style={{width: '1.25rem', height: '1.25rem'}} className="text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          Summary
        </h2>
      </div>

      <div className="flex-fill overflow-auto p-3">
        {/* Summary Stats Cards */}
        <div className="row g-3 mb-4">
          <div className="col-6 col-md-3">
            <div className="card h-100 border-primary">
              <div className="card-body text-center">
                <h3 className="h2 text-primary mb-0">{(analytics.total_sms + analytics.total_mms).toLocaleString()}</h3>
                <small className="text-muted">Total Messages</small>
              </div>
            </div>
          </div>
          <div className="col-6 col-md-3">
            <div className="card h-100 border-success">
              <div className="card-body text-center">
                <h3 className="h2 text-success mb-0">{analytics.total_calls.toLocaleString()}</h3>
                <small className="text-muted">Total Calls</small>
              </div>
            </div>
          </div>
          <div className="col-6 col-md-3">
            <div className="card h-100 border-info">
              <div className="card-body text-center">
                <h3 className="h2 text-info mb-0">{formatDuration(analytics.total_call_duration)}</h3>
                <small className="text-muted">Call Duration</small>
              </div>
            </div>
          </div>
          <div className="col-6 col-md-3">
            <div className="card h-100 border-warning">
              <div className="card-body text-center">
                <h3 className="h2 text-warning mb-0">{Math.round(analytics.avg_message_length)}</h3>
                <small className="text-muted">Avg Chars/Msg</small>
              </div>
            </div>
          </div>
        </div>

        {/* Charts Row 1: Sent/Received + Top Contacts */}
        <div className="row g-3 mb-4">
          {/* Sent vs Received Pie */}
          <div className="col-md-4">
            <div className="card h-100">
              <div className="card-header">Sent vs Received</div>
              <div className="card-body">
                {messageTypeData.length > 0 ? (
                  <ResponsiveContainer width="100%" height={240}>
                    <PieChart>
                      <Pie
                        data={messageTypeData}
                        cx="50%"
                        cy="45%"
                        innerRadius={40}
                        outerRadius={70}
                        paddingAngle={5}
                        dataKey="value"
                        label={({ percent }) => `${(percent * 100).toFixed(0)}%`}
                        labelLine={true}
                      >
                        {messageTypeData.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                        ))}
                      </Pie>
                      <Tooltip formatter={(value, name) => [value.toLocaleString(), name]} />
                      <Legend />
                    </PieChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="text-center text-muted py-5">No message data</div>
                )}
              </div>
            </div>
          </div>

          {/* Top Contacts */}
          <div className="col-md-8">
            <div className="card h-100">
              <div className="card-header">Top Contacts</div>
              <div className="card-body">
                {topContactsData.length > 0 ? (
                  <ResponsiveContainer width="100%" height={200}>
                    <BarChart data={topContactsData} layout="vertical">
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis type="number" />
                      <YAxis
                        type="category"
                        dataKey="displayName"
                        width={120}
                        tick={{ fontSize: 11 }}
                      />
                      <Tooltip
                        formatter={(value) => [value.toLocaleString(), 'Messages']}
                        labelFormatter={(label) => label}
                      />
                      <Bar dataKey="message_count" fill="#0d6efd" name="Messages" />
                    </BarChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="text-center text-muted py-5">No contact data</div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Charts Row 2: Hourly Distribution */}
        <div className="row g-3 mb-4">
          <div className="col-12">
            <div className="card">
              <div className="card-header">Messages by Time of Day</div>
              <div className="card-body">
                {analytics.hourly_distribution && analytics.hourly_distribution.some(h => h.count > 0) ? (
                  <ResponsiveContainer width="100%" height={250}>
                    <BarChart data={analytics.hourly_distribution}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis
                        dataKey="hour"
                        tickFormatter={formatHour}
                        interval={2}
                      />
                      <YAxis />
                      <Tooltip
                        labelFormatter={(hour) => formatHour(hour)}
                        formatter={(value) => [value.toLocaleString(), 'Messages']}
                      />
                      <Bar dataKey="count" fill="#198754" name="Messages" />
                    </BarChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="text-center text-muted py-5">No hourly data</div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Charts Row 3: Daily Trend */}
        <div className="row g-3 mb-4">
          <div className="col-12">
            <div className="card">
              <div className="card-header">Message Trend Over Time</div>
              <div className="card-body">
                {analytics.daily_trend && analytics.daily_trend.length > 0 ? (
                  <ResponsiveContainer width="100%" height={250}>
                    <LineChart data={analytics.daily_trend}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis
                        dataKey="date"
                        tickFormatter={(date) => {
                          const d = new Date(date)
                          return `${d.getMonth() + 1}/${d.getDate()}`
                        }}
                        interval="preserveStartEnd"
                      />
                      <YAxis />
                      <Tooltip
                        labelFormatter={(date) => new Date(date).toLocaleDateString()}
                        formatter={(value) => [value.toLocaleString(), 'Messages']}
                      />
                      <Line
                        type="monotone"
                        dataKey="count"
                        stroke="#0d6efd"
                        strokeWidth={2}
                        dot={false}
                        name="Messages"
                      />
                    </LineChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="text-center text-muted py-5">No trend data</div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Call Statistics */}
        {analytics.total_calls > 0 && (
          <div className="row g-3">
            <div className="col-md-6">
              <div className="card">
                <div className="card-header">Call Breakdown</div>
                <div className="card-body">
                  {callTypeData.length > 0 ? (
                    <ResponsiveContainer width="100%" height={200}>
                      <PieChart>
                        <Pie
                          data={callTypeData}
                          cx="50%"
                          cy="50%"
                          outerRadius={80}
                          dataKey="value"
                          label={({ name, value }) => `${name}: ${value}`}
                        >
                          {callTypeData.map((entry, index) => (
                            <Cell key={`cell-${index}`} fill={COLORS[(index + 2) % COLORS.length]} />
                          ))}
                        </Pie>
                        <Tooltip />
                        <Legend />
                      </PieChart>
                    </ResponsiveContainer>
                  ) : (
                    <div className="text-center text-muted py-5">No call data</div>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default Summary
