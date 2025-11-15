import { useState, useEffect, useRef } from 'react'
import { useLocation } from 'react-router-dom'
import axios from 'axios'
import { format } from 'date-fns'
import LazyMedia from './LazyMedia'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8081/api'

function MessageThread({ conversation, startDate, endDate }) {
  const location = useLocation()
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(false)
  const [highlightedMessageId, setHighlightedMessageId] = useState(null)
  const messageRefs = useRef({})

  useEffect(() => {
    if (conversation) {
      fetchItems()
    } else {
      setItems([])
    }
  }, [conversation, startDate, endDate])

  // Scroll to specific message if messageId is in URL
  useEffect(() => {
    if (items.length > 0) {
      const params = new URLSearchParams(location.search)
      const messageId = params.get('messageId')

      if (messageId) {
        setHighlightedMessageId(messageId)
        if (messageRefs.current[messageId]) {
          const element = messageRefs.current[messageId]

          // Function to wait for media in an element to load
          const waitForMediaInElement = (elem) => {
            const images = Array.from(elem.querySelectorAll('img'))
            const videos = Array.from(elem.querySelectorAll('video'))
            const audios = Array.from(elem.querySelectorAll('audio'))
            const media = [...images, ...videos, ...audios]

            if (media.length === 0) {
              return Promise.resolve()
            }

            const mediaPromises = media.map(mediaElement => {
              if (mediaElement.complete || mediaElement.readyState >= 2) {
                return Promise.resolve()
              }
              return new Promise((resolve) => {
                mediaElement.addEventListener('load', resolve, { once: true })
                mediaElement.addEventListener('loadeddata', resolve, { once: true })
                mediaElement.addEventListener('error', resolve, { once: true })
                setTimeout(resolve, 3000)
              })
            })

            return Promise.all(mediaPromises)
          }

          // Function to perform the scroll
          const scrollToElement = () => {
            element.scrollIntoView({
              behavior: 'smooth',
              block: 'center'
            })
          }

          // Multi-stage scroll approach:
          // 1. Initial scroll to get element near viewport (triggers lazy loading)
          // 2. Wait for lazy-loaded media
          // 3. Final scroll to correct position
          setTimeout(() => {
            // First scroll - instant to trigger lazy loading
            element.scrollIntoView({
              behavior: 'instant',
              block: 'center'
            })

            // Wait a bit for lazy loading to trigger
            setTimeout(() => {
              // Wait for media to load
              waitForMediaInElement(element).then(() => {
                // Final smooth scroll to correct position
                scrollToElement()

                // Re-scroll after a delay to handle any late-loading media
                setTimeout(scrollToElement, 500)
                setTimeout(scrollToElement, 1500)
              })
            }, 200)
          }, 100)
        }
      } else {
        setHighlightedMessageId(null)
      }
    }
  }, [items, location.search])

  // Automatically scroll to the last message when opening a conversation
  useEffect(() => {
    if (items.length > 0) {
      const params = new URLSearchParams(location.search)
      const messageId = params.get('messageId')

      // Only auto-scroll if there's no specific messageId in the URL
      if (!messageId) {
        // Find the last message (not a call) to scroll to
        const lastItem = items[items.length - 1]
        let lastMessageId = null

        // Handle ActivityItem format vs direct Message format
        if (lastItem.type === 'message' && lastItem.message) {
          lastMessageId = lastItem.message.id
        } else if (lastItem.type === 'call') {
          // If last item is a call, find the last message before it
          for (let i = items.length - 1; i >= 0; i--) {
            if (items[i].type === 'message' && items[i].message) {
              lastMessageId = items[i].message.id
              break
            }
          }
        } else if (lastItem.id) {
          // Direct message format
          lastMessageId = lastItem.id
        }

        if (lastMessageId && messageRefs.current[lastMessageId]) {
          const element = messageRefs.current[lastMessageId]

          // Function to wait for media in an element to load
          const waitForMediaInElement = (elem) => {
            const images = Array.from(elem.querySelectorAll('img'))
            const videos = Array.from(elem.querySelectorAll('video'))
            const audios = Array.from(elem.querySelectorAll('audio'))
            const media = [...images, ...videos, ...audios]

            if (media.length === 0) {
              return Promise.resolve()
            }

            const mediaPromises = media.map(mediaElement => {
              if (mediaElement.complete || mediaElement.readyState >= 2) {
                return Promise.resolve()
              }
              return new Promise((resolve) => {
                mediaElement.addEventListener('load', resolve, { once: true })
                mediaElement.addEventListener('loadeddata', resolve, { once: true })
                mediaElement.addEventListener('error', resolve, { once: true })
                setTimeout(resolve, 3000)
              })
            })

            return Promise.all(mediaPromises)
          }

          // Function to perform the scroll
          const scrollToElement = () => {
            element.scrollIntoView({
              behavior: 'instant',
              block: 'end'
            })
          }

          // Scroll to last message after a short delay to ensure rendering is complete
          setTimeout(() => {
            // First scroll to trigger lazy loading if needed
            scrollToElement()

            // Wait for media to load, then scroll again
            setTimeout(() => {
              waitForMediaInElement(element).then(() => {
                scrollToElement()
                // Re-scroll after a delay to handle any late-loading media
                setTimeout(scrollToElement, 300)
              })
            }, 100)
          }, 100)
        }
      }
    }
  }, [items, location.search])

  const fetchItems = async () => {
    setLoading(true)
    try {
      const params = {
        address: conversation.address,
        type: conversation.type
      }
      if (startDate) params.start = startDate.toISOString()
      if (endDate) params.end = endDate.toISOString()

      const response = await axios.get(`${API_BASE}/messages`, { params })
      setItems(response.data || [])
    } catch (error) {
      console.error('Error fetching items:', error)
    } finally {
      setLoading(false)
    }
  }

  const formatTime = (date) => {
    return format(new Date(date), 'MMM d, yyyy h:mm a')
  }

  const formatDuration = (seconds) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  const formatPhoneNumber = (number) => {
    if (!number) return 'Unknown'

    // Handle comma-separated numbers (group conversations)
    if (number.includes(',')) {
      const numbers = number.split(',').map(n => n.trim())
      return numbers.map(n => formatSinglePhoneNumber(n)).join(', ')
    }

    return formatSinglePhoneNumber(number)
  }

  const formatSinglePhoneNumber = (number) => {
    if (!number) return 'Unknown'

    // Remove all non-digit characters
    const cleaned = number.replace(/\D/g, '')

    // Handle 11-digit numbers (e.g., +1 country code)
    if (cleaned.length === 11 && cleaned.startsWith('1')) {
      return `+1 (${cleaned.slice(1, 4)}) ${cleaned.slice(4, 7)}-${cleaned.slice(7)}`
    }

    // Handle 10-digit numbers (US format)
    if (cleaned.length === 10) {
      return `(${cleaned.slice(0, 3)}) ${cleaned.slice(3, 6)}-${cleaned.slice(6)}`
    }

    // Handle other formats - try to format with spaces
    if (cleaned.length > 10) {
      // International format: +XX XXX XXX XXXX
      return `+${cleaned.slice(0, cleaned.length - 10)} ${cleaned.slice(cleaned.length - 10, cleaned.length - 7)} ${cleaned.slice(cleaned.length - 7, cleaned.length - 4)} ${cleaned.slice(cleaned.length - 4)}`
    }

    // Return original if we can't format it nicely
    return number
  }

  const getDisplayName = (conv) => {
    // If we have a valid subject, use it when contact_name is empty, "(Unknown)", or looks like an 8-digit number
    if (conv.subject && shouldDisplaySubject(conv.subject)) {
      if (!conv.contact_name || conv.contact_name === '(Unknown)' || /^\d{8}$/.test(conv.contact_name)) {
        return conv.subject
      }
    }
    // If contact_name is empty, null, or "(Unknown)", use formatted phone number
    if (!conv.contact_name || conv.contact_name === '(Unknown)') {
      return formatPhoneNumber(conv.address)
    }
    return conv.contact_name
  }

  const shouldDisplaySubject = (subject) => {
    if (!subject) return false
    // Filter out protocol buffer/RCS subjects
    if (subject.startsWith('proto:')) return false
    return true
  }

  const getCallTypeInfo = (type) => {
    switch (type) {
      case 1: return { label: 'Incoming', color: 'text-success', bgColor: 'bg-success', icon: '↓' }
      case 2: return { label: 'Outgoing', color: 'text-primary', bgColor: 'bg-primary', icon: '↑' }
      case 3: return { label: 'Missed', color: 'text-danger', bgColor: 'bg-danger', icon: '✕' }
      case 4: return { label: 'Voicemail', color: 'text-info', bgColor: 'bg-info', icon: '⊙' }
      case 5: return { label: 'Rejected', color: 'text-warning', bgColor: 'bg-warning', icon: '✕' }
      case 6: return { label: 'Refused', color: 'text-secondary', bgColor: 'bg-secondary', icon: '✕' }
      default: return { label: 'Call', color: 'text-secondary', bgColor: 'bg-secondary', icon: '○' }
    }
  }

  // Check if conversation is a group conversation
  // Handle both ActivityItem format (items[0].message) and direct Message format (items[0])
  const isGroupConversation = items.length > 0 && (() => {
    const firstItem = items[0]
    // ActivityItem format: check message.addresses
    if (firstItem.type === 'message' && firstItem.message) {
      return firstItem.message.addresses && firstItem.message.addresses.length > 1
    }
    // Direct Message format: check addresses directly
    return firstItem.addresses && firstItem.addresses.length > 1
  })()

  // Get sender display name for a message
  const getSenderDisplayName = (message) => {
    // For received messages, use the sender field if available
    let senderPhone = message.sender

    // If sender is empty, try to extract from addresses array
    // (exclude any number that might be "me" - this is a received message so sender is someone else)
    if (!senderPhone && message.addresses && message.addresses.length > 0) {
      // For now, use the first address as the sender
      // In the future, we could exclude the current user's number
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

  if (!conversation) {
    return (
      <div className="d-flex align-items-center justify-content-center h-100 text-muted">
        <div className="text-center">
          <svg style={{width: '5rem', height: '5rem'}} className="mx-auto mb-3 text-secondary opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
          </svg>
          <p className="h5 text-dark">Select a conversation</p>
          <p className="small mt-2">Choose a conversation from the list to view messages</p>
        </div>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="d-flex align-items-center justify-content-center h-100">
        <div className="text-center">
          <div className="spinner-border text-primary mb-3" role="status" style={{width: '3rem', height: '3rem'}}>
            <span className="visually-hidden">Loading...</span>
          </div>
          <p className="text-muted fw-medium">Loading messages...</p>
        </div>
      </div>
    )
  }

  const isCallLog = conversation.type === 'call'

  return (
    <div className="d-flex flex-column h-100">
      {/* Thread Header */}
      <div className="bg-light border-bottom p-4 shadow-sm">
        <div className="d-flex align-items-center gap-3">
          <div className="p-3 rounded-circle bg-primary bg-gradient shadow">
            {isCallLog ? (
              <svg style={{width: '1.5rem', height: '1.5rem'}} className="text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
              </svg>
            ) : (
              <svg style={{width: '1.5rem', height: '1.5rem'}} className="text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
              </svg>
            )}
          </div>
          <div>
            <h2 className="h4 fw-bold mb-1">
              {getDisplayName(conversation)}
            </h2>
            {/* Display phone numbers for conversations with addresses */}
            {!isCallLog && items.length > 0 && (() => {
              const firstItem = items[0]
              // Get addresses from either ActivityItem.message or direct Message
              const addresses = (firstItem.type === 'message' && firstItem.message)
                ? firstItem.message.addresses
                : firstItem.addresses
              return addresses && addresses.length > 0 && (
                <div className="small text-muted mb-2">
                  {addresses.map((addr, idx) => (
                    <span key={idx}>
                      {formatPhoneNumber(addr)}
                      {idx < addresses.length - 1 ? ', ' : ''}
                    </span>
                  ))}
                </div>
              )
            })()}
            <div>
              <span className="badge bg-primary">
                {items.length} {isCallLog ? 'call' : 'message'}{items.length !== 1 ? 's' : ''}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-fill overflow-auto p-4 bg-light">
        {isCallLog ? (
          // Call Log View
          <div className="d-flex flex-column gap-3">
            {items.map((call) => {
              const typeInfo = getCallTypeInfo(call.type)
              return (
                <div
                  key={call.id}
                  className="card shadow-sm border-2"
                >
                  <div className="card-body">
                    <div className="d-flex align-items-center justify-content-between">
                      <div className="d-flex align-items-center gap-3">
                        <div className={`p-3 rounded-circle ${typeInfo.bgColor} bg-opacity-10`}>
                          <span className={`fs-4 ${typeInfo.color}`}>
                            {typeInfo.icon}
                          </span>
                        </div>
                        <div>
                          <div className={`fw-semibold ${typeInfo.color}`}>
                            {typeInfo.label} Call
                          </div>
                          <div className="small text-muted mt-1 d-flex align-items-center gap-1">
                            <svg style={{width: '1rem', height: '1rem'}} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                            {formatTime(call.date)}
                          </div>
                        </div>
                      </div>
                      <div className="text-end">
                        <div className="h5 fw-bold mb-0">
                          {formatDuration(call.duration)}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        ) : (
          // Unified Message and Call View
          <div className="d-flex flex-column gap-1">
            {items.map((item) => {
              // Check if this is an ActivityItem (has type field) or a direct Message
              const isActivityItem = item.type === 'message' || item.type === 'call'
              const isCall = isActivityItem && item.type === 'call'
              const message = isActivityItem ? item.message : item
              const call = isActivityItem ? item.call : null

              if (isCall && call) {
                // Compact call representation - inline with messages
                const typeInfo = getCallTypeInfo(call.type)
                return (
                  <div key={`call-${call.id}`} className="d-flex justify-content-center my-1">
                    <div className="badge bg-light text-dark border px-3 py-2 d-flex align-items-center gap-2" style={{fontSize: '0.75rem'}}>
                      <span className={typeInfo.color} style={{fontSize: '1rem'}}>{typeInfo.icon}</span>
                      <span className={`fw-semibold ${typeInfo.color}`}>{typeInfo.label} call</span>
                      <span className="text-muted">·</span>
                      <span className="text-muted">{formatTime(call.date)}</span>
                      {call.duration > 0 && (
                        <>
                          <span className="text-muted">·</span>
                          <span className="text-muted">{formatDuration(call.duration)}</span>
                        </>
                      )}
                    </div>
                  </div>
                )
              }

              // Message rendering
              if (!message) return null

              const isSent = message.type === 2
              const isHighlighted = highlightedMessageId === String(message.id)
              const showSenderLabel = isGroupConversation && !isSent

              return (
                <div
                  key={message.id}
                  className={`d-flex ${isSent ? 'justify-content-end' : 'justify-content-start'}`}
                >
                  <div style={{ maxWidth: '70%' }}>
                    {/* Sender label for received messages in group conversations */}
                    {showSenderLabel && (
                      <div className="small text-muted mb-1 ms-2" style={{ fontSize: '0.7rem' }}>
                        {getSenderDisplayName(message)}
                      </div>
                    )}
                    <div
                      ref={(el) => (messageRefs.current[message.id] = el)}
                      className={`card shadow-sm ${
                        isSent
                          ? 'bg-primary text-white'
                          : 'bg-white'
                      } ${
                        isHighlighted
                          ? 'border-warning border-3'
                          : 'border-2'
                      }`}
                      style={{
                        padding: '0.5em',
                        position: 'relative'
                      }}
                    >
                      <div className="card-body py-1 px-2">
                        {message.body && (
                          <div style={{whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontSize: '0.875rem', lineHeight: '1.3'}}>
                            {message.body}
                          </div>
                        )}
                        {message.media_type && (
                          <LazyMedia
                            messageId={message.id}
                            mediaType={message.media_type}
                            className="mt-1"
                            alt="MMS attachment"
                          />
                        )}
                        <div
                          className={`mt-1 d-flex align-items-center gap-1 ${
                            isSent ? 'text-white-50' : 'text-muted'
                          }`}
                          style={{fontSize: '0.75rem'}}
                        >
                          <svg style={{width: '0.7rem', height: '0.7rem'}} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                          </svg>
                          {formatTime(message.date)}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

export default MessageThread
