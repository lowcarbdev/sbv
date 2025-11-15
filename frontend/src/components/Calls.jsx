import { useState, useEffect, useRef, useCallback } from 'react'
import axios from 'axios'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8081/api'
const PAGE_SIZE = 50

function Calls({ startDate, endDate }) {
  const [calls, setCalls] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const [offset, setOffset] = useState(0)
  const observerTarget = useRef(null)
  const scrollContainerRef = useRef(null)

  // Reset when date range changes
  useEffect(() => {
    setCalls([])
    setOffset(0)
    setHasMore(true)
    fetchCalls(0, false)
  }, [startDate, endDate])

  const fetchCalls = async (currentOffset, append = false) => {
    if (append) {
      setLoadingMore(true)
    } else {
      setLoading(true)
    }

    try {
      const params = {
        limit: PAGE_SIZE,
        offset: currentOffset
      }
      if (startDate) params.start = startDate.toISOString()
      if (endDate) params.end = endDate.toISOString()

      const response = await axios.get(`${API_BASE}/calls`, { params })
      const newCalls = response.data || []

      // If we got fewer items than the page size, we've reached the end
      if (newCalls.length < PAGE_SIZE) {
        setHasMore(false)
      }

      if (append) {
        setCalls(prev => [...prev, ...newCalls])
      } else {
        setCalls(newCalls)
      }
    } catch (error) {
      console.error('Error fetching calls:', error)
    } finally {
      setLoading(false)
      setLoadingMore(false)
    }
  }

  const loadMore = useCallback(() => {
    if (!loadingMore && hasMore) {
      const newOffset = offset + PAGE_SIZE
      setOffset(newOffset)
      fetchCalls(newOffset, true)
    }
  }, [offset, loadingMore, hasMore])

  // Set up intersection observer for infinite scroll
  useEffect(() => {
    if (!scrollContainerRef.current || !observerTarget.current) {
      return
    }

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loadingMore) {
          loadMore()
        }
      },
      {
        root: scrollContainerRef.current,
        rootMargin: '100px',
        threshold: 0.1
      }
    )

    observer.observe(observerTarget.current)

    return () => {
      observer.disconnect()
    }
  }, [loadMore, hasMore, loadingMore, calls])

  const formatCallType = (type) => {
    switch (type) {
      case 1: return { label: 'Incoming call', icon: '📞', color: 'success' }
      case 2: return { label: 'Outgoing call', icon: '📱', color: 'primary' }
      case 3: return { label: 'Missed call', icon: '📵', color: 'danger' }
      case 4: return { label: 'Voicemail', icon: '🎙️', color: 'info' }
      case 5: return { label: 'Rejected call', icon: '🚫', color: 'warning' }
      case 6: return { label: 'Refused call', icon: '❌', color: 'danger' }
      default: return { label: 'Call', icon: '📞', color: 'secondary' }
    }
  }

  const formatDuration = (seconds) => {
    if (seconds < 60) return `${seconds}s`
    const minutes = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${minutes}m ${secs}s`
  }

  const formatPhoneNumber = (phoneNumber) => {
    if (!phoneNumber) return ''

    // Remove any non-numeric characters except leading +
    let cleaned = phoneNumber.replace(/[^\d+]/g, '')

    // Handle +1 prefix (US numbers)
    if (cleaned.startsWith('+1') && cleaned.length === 12) {
      // Format as +1 (XXX) XXX-XXXX
      return `+1 (${cleaned.slice(2, 5)}) ${cleaned.slice(5, 8)}-${cleaned.slice(8)}`
    }

    // Handle numbers with + country code
    if (cleaned.startsWith('+')) {
      return cleaned // Return international numbers as-is
    }

    // Handle 11-digit numbers starting with 1 (US numbers without +)
    if (cleaned.length === 11 && cleaned.startsWith('1')) {
      return `+1 (${cleaned.slice(1, 4)}) ${cleaned.slice(4, 7)}-${cleaned.slice(7)}`
    }

    // Handle 10-digit US numbers
    if (cleaned.length === 10) {
      return `+1 (${cleaned.slice(0, 3)}) ${cleaned.slice(3, 6)}-${cleaned.slice(6)}`
    }

    // Return as-is if format doesn't match
    return phoneNumber
  }

  const formatDate = (dateStr) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now - date
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

    const timeStr = date.toLocaleTimeString('en-US', {
      hour: 'numeric',
      minute: '2-digit',
      hour12: true
    })

    if (diffDays === 0) return `Today at ${timeStr}`
    if (diffDays === 1) return `Yesterday at ${timeStr}`
    if (diffDays < 7) return date.toLocaleDateString('en-US', {
      weekday: 'long',
      hour: 'numeric',
      minute: '2-digit',
      hour12: true
    })

    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined,
      hour: 'numeric',
      minute: '2-digit',
      hour12: true
    })
  }

  if (loading) {
    return (
      <div className="h-100 d-flex align-items-center justify-content-center">
        <div className="text-center">
          <div className="spinner-border text-primary mb-3" role="status">
            <span className="visually-hidden">Loading...</span>
          </div>
          <p className="text-muted">Loading calls...</p>
        </div>
      </div>
    )
  }

  if (calls.length === 0) {
    return (
      <div className="h-100 d-flex align-items-center justify-content-center">
        <div className="text-center">
          <svg style={{width: '4rem', height: '4rem'}} className="text-muted mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
          </svg>
          <p className="text-muted">No calls found</p>
        </div>
      </div>
    )
  }

  return (
    <div className="h-100 d-flex flex-column">
      <div className="bg-light border-bottom p-3">
        <h2 className="h5 mb-0 d-flex align-items-center gap-2">
          <svg style={{width: '1.25rem', height: '1.25rem'}} className="text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
          </svg>
          Call History
          <span className="badge bg-primary ms-auto">{calls.length} calls</span>
        </h2>
      </div>

      <div ref={scrollContainerRef} className="flex-fill overflow-auto p-3">
        <div className="container-fluid">
          {calls.map((call) => {
            const callType = formatCallType(call.type)
            const formattedNumber = formatPhoneNumber(call.number)
            const displayName = call.contact_name || formattedNumber

            return (
              <div key={`call-${call.id}`} className="card mb-2 shadow-sm border-start border-4" style={{borderLeftColor: `var(--bs-${callType.color})`}}>
                <div className="card-body py-2 px-3">
                  <div className="d-flex justify-content-between align-items-start">
                    <div className="d-flex align-items-center gap-2">
                      <div style={{fontSize: '1.25rem'}}>{callType.icon}</div>
                      <div>
                        <h6 className="mb-0">
                          {displayName}
                        </h6>
                        <small className="text-muted">{formattedNumber}</small>
                      </div>
                    </div>
                    <div className="text-end">
                      <span className={`badge bg-${callType.color}`}>{callType.label}</span>
                      <br />
                      <small className="text-muted">{formatDate(call.date)}</small>
                    </div>
                  </div>
                  {call.duration > 0 && (
                    <div className="mt-1">
                      <small className="text-muted">
                        <svg style={{width: '1rem', height: '1rem'}} className="me-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        Duration: {formatDuration(call.duration)}
                      </small>
                    </div>
                  )}
                </div>
              </div>
            )
          })}

          {/* Infinite scroll trigger */}
          <div ref={observerTarget} style={{ height: '20px' }} />

          {/* Loading more indicator */}
          {loadingMore && (
            <div className="text-center py-3">
              <div className="spinner-border spinner-border-sm text-primary" role="status">
                <span className="visually-hidden">Loading more...</span>
              </div>
              <p className="small text-muted mt-2 mb-0">Loading more calls...</p>
            </div>
          )}

          {/* End of results indicator */}
          {!hasMore && calls.length > 0 && (
            <div className="text-center py-3">
              <small className="text-muted">No more calls to load</small>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default Calls
