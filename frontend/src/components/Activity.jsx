import { useState, useEffect, useRef, useCallback } from 'react'
import axios from 'axios'
import LazyMedia from './LazyMedia'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8081/api'
const PAGE_SIZE = 50

function Activity({ startDate, endDate }) {
  const [activities, setActivities] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const [offset, setOffset] = useState(0)
  const observerTarget = useRef(null)
  const scrollContainerRef = useRef(null)

  // Reset when date range changes
  useEffect(() => {
    setActivities([])
    setOffset(0)
    setHasMore(true)
    fetchActivity(0, false)
  }, [startDate, endDate])

  const fetchActivity = async (currentOffset, append = false) => {
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

      const response = await axios.get(`${API_BASE}/activity`, { params })
      const newActivities = response.data || []

      // If we got fewer items than the page size, we've reached the end
      if (newActivities.length < PAGE_SIZE) {
        setHasMore(false)
      }

      if (append) {
        setActivities(prev => [...prev, ...newActivities])
      } else {
        setActivities(newActivities)
      }
    } catch (error) {
      console.error('Error fetching activity:', error)
    } finally {
      setLoading(false)
      setLoadingMore(false)
    }
  }

  const loadMore = useCallback(() => {
    console.log('loadMore called:', { loadingMore, hasMore, offset })
    if (!loadingMore && hasMore) {
      const newOffset = offset + PAGE_SIZE
      setOffset(newOffset)
      fetchActivity(newOffset, true)
    }
  }, [offset, loadingMore, hasMore])

  // Set up intersection observer for infinite scroll
  useEffect(() => {
    // Make sure both refs are available
    if (!scrollContainerRef.current || !observerTarget.current) {
      console.log('Refs not ready:', { scroll: !!scrollContainerRef.current, target: !!observerTarget.current })
      return
    }

    console.log('Setting up IntersectionObserver', { hasMore, loadingMore, activitiesCount: activities.length })

    const observer = new IntersectionObserver(
      (entries) => {
        console.log('Observer callback fired', {
          isIntersecting: entries[0].isIntersecting,
          hasMore,
          loadingMore
        })
        if (entries[0].isIntersecting && hasMore && !loadingMore) {
          console.log('Intersection detected, loading more...')
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
  }, [loadMore, hasMore, loadingMore, activities])

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

    // Handle comma-separated numbers (group conversations)
    if (phoneNumber.includes(',')) {
      const numbers = phoneNumber.split(',').map(n => n.trim())
      return numbers.map(n => formatSinglePhoneNumber(n)).join(', ')
    }

    return formatSinglePhoneNumber(phoneNumber)
  }

  const formatSinglePhoneNumber = (phoneNumber) => {
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

  const getMessageTypeLabel = (type) => {
    switch (type) {
      case 1: return { label: 'Received', color: 'primary' }
      case 2: return { label: 'Sent', color: 'success' }
      case 3: return { label: 'Draft', color: 'secondary' }
      case 4: return { label: 'Outbox', color: 'warning' }
      case 5: return { label: 'Failed', color: 'danger' }
      case 6: return { label: 'Queued', color: 'info' }
      default: return { label: 'Message', color: 'secondary' }
    }
  }

  const shouldDisplaySubject = (subject) => {
    if (!subject) return false
    // Filter out protocol buffer/RCS subjects
    if (subject.startsWith('proto:')) return false
    return true
  }

  // Get sender display name for a message in group conversations
  const getSenderDisplayName = (message) => {
    // For received messages, use the sender field if available
    let senderPhone = message.sender

    // If sender is empty, try to extract from addresses array
    if (!senderPhone && message.addresses && message.addresses.length > 0) {
      // Use the first address as the sender
      senderPhone = message.addresses[0]
    }

    // If sender contains comma-separated numbers (shouldn't happen, but handle it),
    // extract only the first one
    if (senderPhone && senderPhone.includes(',')) {
      senderPhone = senderPhone.split(',')[0].trim()
    }

    if (!senderPhone) return 'Unknown'

    // Format as a single phone number (not as a group)
    return formatSinglePhoneNumber(senderPhone)
  }

  if (loading) {
    return (
      <div className="h-100 d-flex align-items-center justify-content-center">
        <div className="text-center">
          <div className="spinner-border text-primary mb-3" role="status">
            <span className="visually-hidden">Loading...</span>
          </div>
          <p className="text-muted">Loading activity...</p>
        </div>
      </div>
    )
  }

  if (activities.length === 0) {
    return (
      <div className="h-100 d-flex align-items-center justify-content-center">
        <div className="text-center">
          <svg style={{width: '4rem', height: '4rem'}} className="text-muted mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <p className="text-muted">No activity found</p>
        </div>
      </div>
    )
  }

  return (
    <div className="h-100 d-flex flex-column">
      <div className="bg-light border-bottom p-3">
        <h2 className="h5 mb-0 d-flex align-items-center gap-2">
          <svg style={{width: '1.25rem', height: '1.25rem'}} className="text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          Activity Timeline
          <span className="badge bg-primary ms-auto">{activities.length} items</span>
        </h2>
      </div>

      <div ref={scrollContainerRef} className="flex-fill overflow-auto p-3">
        <div className="container-fluid">
          {activities.map((activity, index) => {
            if (activity.type === 'message' && activity.message) {
              const msg = activity.message
              const msgType = getMessageTypeLabel(msg.type)

              // For MMS with multiple recipients, use the addresses array; otherwise use the single address
              let displayAddress
              if (msg.addresses && msg.addresses.length > 0) {
                // Format each address and join with commas
                displayAddress = msg.addresses.map(addr => formatPhoneNumber(addr)).join(', ')
              } else {
                // Fall back to the single address field
                displayAddress = formatPhoneNumber(activity.address)
              }

              const displayName = activity.contact_name || displayAddress

              // Check if this is a group conversation
              const isGroupConversation = msg.addresses && msg.addresses.length > 1
              const isSent = msg.type === 2
              const showSenderLabel = isGroupConversation && !isSent

              // Debug logging for ALL messages to understand what we're receiving
              console.log('Message received:', {
                id: msg.id,
                addresses: msg.addresses,
                addressesType: typeof msg.addresses,
                addressesLength: msg.addresses?.length,
                sender: msg.sender,
                address: msg.address,
                type: msg.type,
                isSent,
                isGroupConversation,
                showSenderLabel,
                body: msg.body?.substring(0, 30)
              })

              return (
                <div key={`msg-${msg.id}`} className="card mb-2 shadow-sm" style={{padding: '0.5rem'}}>
                  <div className="card-body py-2 px-3">
                    <div className="d-flex justify-content-between align-items-start mb-1">
                      <div className="d-flex align-items-center gap-2">
                        <svg style={{width: '1.25rem', height: '1.25rem'}} className="text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                        </svg>
                        <div>
                          <h6 className="mb-0">
                            {displayName}
                          </h6>
                          <small className="text-muted">{displayAddress}</small>
                        </div>
                      </div>
                      <div className="text-end">
                        <span className={`badge bg-${msgType.color}`}>{msgType.label}</span>
                        <br />
                        <small className="text-muted">{formatDate(activity.date)}</small>
                      </div>
                    </div>

                    {/* Sender label for received messages in group conversations */}
                    {showSenderLabel && (
                      <div className="mb-1 ps-2">
                        <small className="text-muted fw-semibold">
                          From: {getSenderDisplayName(msg)}
                        </small>
                      </div>
                    )}

                    {shouldDisplaySubject(msg.subject) && (
                      <div className="mb-1">
                        <strong>Subject:</strong> {msg.subject}
                      </div>
                    )}

                    {msg.body && (
                      <p className="card-text mb-1">{msg.body}</p>
                    )}

                    {msg.media_type && (
                      <LazyMedia
                        messageId={msg.id}
                        mediaType={msg.media_type}
                        className="mt-1"
                        alt="MMS attachment"
                      />
                    )}
                  </div>
                </div>
              )
            } else if (activity.type === 'call' && activity.call) {
              const call = activity.call
              const callType = formatCallType(call.type)
              const formattedAddress = formatPhoneNumber(activity.address)
              const displayName = activity.contact_name || formattedAddress

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
                          <small className="text-muted">{formattedAddress}</small>
                        </div>
                      </div>
                      <div className="text-end">
                        <span className={`badge bg-${callType.color}`}>{callType.label}</span>
                        <br />
                        <small className="text-muted">{formatDate(activity.date)}</small>
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
            }
            return null
          })}

          {/* Infinite scroll trigger */}
          <div ref={observerTarget} style={{ height: '20px' }} />

          {/* Loading more indicator */}
          {loadingMore && (
            <div className="text-center py-3">
              <div className="spinner-border spinner-border-sm text-primary" role="status">
                <span className="visually-hidden">Loading more...</span>
              </div>
              <p className="small text-muted mt-2 mb-0">Loading more activities...</p>
            </div>
          )}

          {/* End of results indicator */}
          {!hasMore && activities.length > 0 && (
            <div className="text-center py-3">
              <small className="text-muted">No more activities to load</small>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default Activity
